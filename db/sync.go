package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/util/noms/reachable"
)

func (db *DB) Sync(remote spec.Spec) error {
	remoteDB, err := Load(remote, "") // TODO: do something about empty origin
	if err != nil {
		return err
	}

	progress := make(chan datas.PullProgress)
	go func() {
		for p := range progress {
			fmt.Println("pull progress", p)
		}
	}()

	// 1: Push client head to server
	datas.Pull(db.noms, remoteDB.noms, types.NewRef(db.head.Original), progress)

	// 2: Merge client changes into server state
	// TODO: This will become an RPC to a remote server that will do this step on its side.
	localHead := db.head
	rebased, err := remoteDB.handleSync(localHead)
	if err != nil {
		return err
	}

	// 3: Save the new remote state - primarily to avoid re-downloading it in the future and for debugging purposes.
	_, err = db.noms.SetHead(db.noms.GetDataset(remote_dataset), types.NewRef(rebased.Original))
	if err != nil {
		return err
	}

	// 4: Rebase any new local changes from between 1 and 3.
	rebased, err = rebase(db, types.NewRef(rebased.Original), db.head)
	if err != nil {
		return err
	}

	// 5: Commit new local head.
	_, err = db.noms.FastForward(db.noms.GetDataset(local_dataset), db.noms.WriteValue(rebased.Original))
	return err
}

func (db *DB) handleSync(commit Commit) (newHead Commit, err error) {
	// TODO: This needs to be done differently.

	// Noms already tracks exactly which chunks have been "flushed" to the chunkstore.
	// The difference is that there is no guarantee that those chunks are reachable from
	// a particular head.
	// Alternately it probably needs to be global and kept updated.
	// Or alternate-alternately, it could be implemented so that it is crawled incrementally
	reachable := reachable.New(db.noms)
	err = validate(db, reachable, commit)
	if err != nil {
		return Commit{}, err
	}
	rebased, err := rebase(db, types.NewRef(db.head.Original), commit)
	if err != nil {
		return Commit{}, err
	}
	_, err = db.noms.FastForward(db.noms.GetDataset(local_dataset), db.noms.WriteValue(rebased.Original))
	if err != nil {
		return Commit{}, err
	}
	return rebased, nil
}
