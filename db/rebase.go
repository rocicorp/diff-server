package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/util/noms/reachable"
)

func rebase(db *DB, reachable *reachable.Set, onto types.Ref, date datetime.DateTime, commit Commit) (rebased Commit, err error) {
	if reachable.Has(commit.Original.Hash()) {
		var r Commit
		err = marshal.Unmarshal(onto.TargetValue(db.noms), &r)
		if err != nil {
			return Commit{}, err
		}
		return r, nil
	}

	oldBasis, err := commit.Basis(db.noms)
	if err != nil {
		return Commit{}, err
	}

	if onto.TargetHash() == oldBasis.Original.Hash() {
		return commit, nil
	}

	newBasis, err := rebase(db, reachable, onto, date, oldBasis)
	if err != nil {
		return Commit{}, err
	}

	var newBundle, newData types.Ref

	switch commit.Type() {
	case CommitTypeTx:
		newBundle, newData, err = db.execImpl(types.NewRef(newBasis.Original), commit.Meta.Tx.Bundle(db.noms), commit.Meta.Tx.Name, commit.Meta.Tx.Args)
		if err != nil {
			return Commit{}, err
		}
		break

	case CommitTypeReorder:
		target, err := commit.FinalReorderTarget(db.noms)
		if err != nil {
			return Commit{}, err
		}
		newBundle, newData, err = db.execImpl(types.NewRef(newBasis.Original), target.Meta.Tx.Bundle(db.noms), target.Meta.Tx.Name, target.Meta.Tx.Args)
		if err != nil {
			return Commit{}, err
		}

	default:
		return Commit{}, fmt.Errorf("Cannot rebase commit of type %s: %s: %s", commit.Type(), commit.Original.Hash(), types.EncodedValue(commit.Original))
	}

	newCommit := makeReorder(db.noms, types.NewRef(newBasis.Original), db.origin, date, types.NewRef(commit.Original), newBundle, newData)
	db.noms.WriteValue(newCommit.Original)
	return newCommit, nil
}
