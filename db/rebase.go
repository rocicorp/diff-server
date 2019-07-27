package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/attic-labs/noms/go/types"
)

func rebase(db *DB, onto types.Ref, commit Commit) (rebased Commit, err error) {
	if onto.TargetHash() == commit.Original.Hash() {
		return Commit{}, nil
	}

	oldBasis, err := commit.Basis(db.noms)
	if err != nil {
		return Commit{}, err
	}
	newBasis, err := rebase(db, onto, oldBasis)
	if err != nil {
		return Commit{}, err
	}

	var newBundle, newData types.Ref
	d := datetime.Now()

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

	newCommit := makeReorder(db.noms, types.NewRef(newBasis.Original), db.origin, d, types.NewRef(commit.Original), newBundle, newData)
	db.noms.WriteValue(newCommit.Original)
	return newCommit, nil
}
