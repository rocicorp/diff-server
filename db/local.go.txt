package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/exec"
	"github.com/aboodman/replicant/util/history"
)

// apply(parsed, db, cache):
//   if cache.Has(parsed): return
//   apply(parsed.Basis()) (recurse)
//   commit = replay(parsed, db.LocalHead(), trust)
//   writeValue(commit)
//   db.setLocalHead(commit)
//
// replay(parsed, db, cache):
//
//   switch parsed.type():
//     case tx:
//       val = run(parsed.meta.tx.fn, parsed.meta.tx.fn.args, db.localHead())
//     case apply:
//	     val = replay(parsed.target(), db, cache)
//     commit = prepareCommit(...)
//   if parsed.basis() != db.localHead()
//
// // validate entire new client tree before applying, if desired
// validate(commit):
//   if cache.Has(commit): return true
//   for _, p := range parents: validate(p)
//   v = replay(commit, commit.Basis())
//   return v === commit
//

// LocalDest implements a sync destination where the merge function runs locally.
// The actual destination database can be remote, but if access to it is highly latent,
// then this will be much slower than running the merge function at the destination.
type LocalDest struct {
	db  *DB
	now func() time.Time
}

func NewLocalDest(sp spec.Spec) (LocalDest, error) {
	db, err := Load(sp)
	if err != nil {
		return LocalDest{}, err
	}
	return LocalDest{db: db}, nil
}

func (ld *LocalDest) Merge(clientHash hash.Hash) (merged types.Ref, err error) {
	clientHead := ld.db.Noms().ReadValue(clientHash)
	if clientHead == nil {
		return types.Ref{}, errors.New("clientHead does not exist")
	}

	localHead := ld.db.Head()
	mc, err := ld.merge(clientHead, localHead)
	if err != nil {
		return types.Ref{}, err
	}
	r, err := ld.db.Commit(mc)
	if err != nil {
		return types.Ref{}, err
	}
	return r, nil
}

func (ld *LocalDest) merge(clientHead, localHead types.Value) (Commit, error) {
	cache := history.NewCache(ld.db.Noms())
	if localHead != nil {
		err := cache.Populate(localHead.Hash())
		if err != nil {
			return Commit{}, err
		}
	}

	// TODO: validate() client head too, at least in debug mode

	r, err := ld.apply(clientHead, cache)
	if err != nil {
		return Commit{}, fmt.Errorf("Could not apply client head: %s", err.Error())
	}

	return r, nil
}

func (ld *LocalDest) apply(clientHead types.Value, cache *history.Cache) (Commit, error) {
	var parsed Commit
	err := marshal.Unmarshal(clientHead, &parsed)
	if err != nil {
		return Commit{}, err
	}

	// base case - already committed
	// TODO - should we be adding to the cache during merge somewhere?
	if cache.Has(clientHead.Hash()) {
		return parsed, nil
	}

	claimedBasis := parsed.Basis()
	if !claimedBasis.IsZeroValue() {
		_, err := ld.apply(claimedBasis.TargetValue(ld.db.Noms()), cache)
		if err != nil {
			return Commit{}, err
		}
	}
	basis := ld.db.Head()
	r, err := ld.replay(parsed, types.NewRef(basis))
	if err != nil {
		return Commit{}, err
	}
	return r, nil
}

func (ld *LocalDest) replay(commit Commit, basis types.Ref) (Commit, error) {
	// TODO: if trust, and basis same as onto, skip forward
	c := commit
	for {
		switch c.Type() {
		case CommitTypeTx:
			{
				code := c.Meta.Tx.Code.TargetValue(ld.db.Noms()).(types.Blob)
				err := exec.Run(ld.db, code.Reader(), c.Meta.Tx.Name, c.Meta.Tx.Args)
				if err != nil {
					return Commit{}, err
				}
				var r Commit
				if basis.Equals(commit.Basis()) {
					r = commit
				} else {
					now := time.Now()
					if ld.now != nil {
						now = ld.now()
					}
					r, err = ld.db.MakeReorder(commit, datetime.DateTime{now})
					if err != nil {
						return Commit{}, err
					}
				}
				_, err = ld.db.Commit(r)
				if err != nil {
					return Commit{}, err
				}
				return r, nil
			}
		case CommitTypeReorder, CommitTypeReject:
			{
				target, err := c.TargetCommit(ld.db.Noms())
				if err != nil {
					return Commit{}, err
				}
				c = target
			}
		default:
			return Commit{}, fmt.Errorf("Unknown commit type for commit: %s", commit.Original.Hash())
		}
	}
}
