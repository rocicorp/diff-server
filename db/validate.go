package db

import (
	"fmt"

	"github.com/aboodman/replicant/util/noms/diff"
	"github.com/aboodman/replicant/util/noms/reachable"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
)

func validate(db *DB, reachable *reachable.Set, commit Commit) error {
	if reachable.Has(commit.Original.Hash()) {
		return nil
	}

	for _, c := range commit.Parents {
		var p Commit
		err := marshal.Unmarshal(c.TargetValue(db.noms), &p)
		if err != nil {
			return err
		}
		err = validate(db, reachable, p)
		if err != nil {
			return err
		}
	}

	var replayed Commit
	switch commit.Type() {
	case CommitTypeTx:
		newBundle, newData, err := db.execImpl(commit.BasisRef(), commit.Meta.Tx.Bundle(db.noms), commit.Meta.Tx.Name, commit.Meta.Tx.Args)
		if err != nil {
			return err
		}
		replayed = makeTx(db.noms, commit.BasisRef(), commit.Meta.Tx.Origin, commit.Meta.Tx.Date, commit.Meta.Tx.Code, commit.Meta.Tx.Name, commit.Meta.Tx.Args, newBundle, newData)
		break

	case CommitTypeReorder:
		target, err := commit.FinalReorderTarget(db.noms)
		if err != nil {
			return err
		}
		newBundle, newData, err := db.execImpl(commit.BasisRef(), target.Meta.Tx.Bundle(db.noms), target.Meta.Tx.Name, target.Meta.Tx.Args)
		if err != nil {
			return err
		}
		replayed = makeReorder(db.noms, commit.BasisRef(), commit.Meta.Reorder.Origin, commit.Meta.Reorder.Date, types.NewRef(target.Original), newBundle, newData)

	case CommitTypeReject:
		b, err := commit.Basis(db.noms)
		if err != nil {
			return err
		}
		replayed = makeReject(db.noms, commit.BasisRef(), commit.Meta.Reject.Origin, commit.Meta.Reject.Date, commit.Meta.Reject.Subject, commit.Meta.Reject.Reason, b.Value.Code, b.Value.Data)

	case CommitTypeGenesis:
		replayed = makeGenesis(db.noms)
	}

	if !replayed.Original.Equals(commit.Original) {
		return fmt.Errorf("Invalid commit %s: diff: %s", replayed.Original.Hash(), diff.Diff(commit.Original, replayed.Original))
	}

	return nil
}
