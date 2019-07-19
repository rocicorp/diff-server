package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
)

// destination is a place that we can sync to, typically the Replicant
// Server for the group the client is a part of.
//
// The clientHead must already exist at the destination before calling
// this, for example by using Noms' datas.Pull().
//
// The resulting merged head references data at the destination. That
// data must be pulled back to the client before it can be used there.
type destination interface {
	Merge(clientHead hash.Hash) (mergedHead hash.Hash, err error)
}

// DoSync implements bidirectional replication and conflict resolution between
// the src (the client) and dest (the server). See the Replicant design
// doc for details.
func (src *DB) Sync(dest spec.Spec) (types.Ref, error) {
	destination, err := NewLocalDest(dest)
	if err != nil {
		return types.Ref{}, err
	}

	srcHeadRef := types.NewRef(src.Head())

	// 1: Push client local head to server
	err = pull(src.Noms(), dest.GetDatabase(), srcHeadRef)
	if err != nil {
		return types.Ref{}, err
	}

	// 2: Merge on server
	merged, err := destination.Merge(srcHeadRef.TargetHash())
	if err != nil {
		return types.Ref{}, err
	}

	// 3: Pull merged head back to client
	err = pull(dest.GetDatabase(), src.Noms(), merged)
	if err != nil {
		return types.Ref{}, err
	}

	// 4: Commit merged head to client remote branch
	_, err = src.Noms().FastForward(src.Noms().GetDataset(REMOTE_DATASET), merged)
	if err != nil {
		return types.Ref{}, err
	}

	// 5: Merge any changes on client with merged changes from server
	fork, err := src.Fork(merged.TargetHash())
	if err != nil {
		return types.Ref{}, err
	}
	destination = LocalDest{db: fork}
	merged, err = destination.Merge(src.Head().Hash())
	if err != nil {
		return types.Ref{}, err
	}

	return merged, nil
}

func pull(src, sink datas.Database, head types.Ref) error {
	pc := make(chan datas.PullProgress)
	go func() {
		for p := range pc {
			fmt.Println(p)
		}
	}()

	// TODO: It would be more efficient to send just the commit struct and meta, but *not* value,
	// and just let server recalculate them, since it has to validate anyway. I guess this would mean push() moves
	// into the Merge() interface.
	datas.Pull(src, sink, head, pc)
	return nil
}
