package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/util/noms/diff"
)

func validate(db *DB, commit Commit, forkPoint types.Ref) (err error) {
	if forkPoint.IsZeroValue() {
		forkPoint, err = commonAncestor(db.head.Ref(), commit.Ref(), db.Noms())
		if err != nil {
			return err
		}
	}

	if commit.Ref().Equals(forkPoint) {
		return nil
	}

	for _, c := range commit.Parents {
		var p Commit
		err = marshal.Unmarshal(c.TargetValue(db.noms), &p)
		if err != nil {
			return err
		}
		err = validate(db, p, forkPoint)
		if err != nil {
			return err
		}
	}

	var replayed Commit
	switch commit.Type() {
	case CommitTypeTx:
		newBundle, newData, _, _, err := db.execImpl(commit.BasisRef(), commit.Meta.Tx.Bundle(db.noms), commit.Meta.Tx.Name, commit.Meta.Tx.Args)
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
		newBundle, newData, _, _, err := db.execImpl(commit.BasisRef(), target.Meta.Tx.Bundle(db.noms), target.Meta.Tx.Name, target.Meta.Tx.Args)
		if err != nil {
			return err
		}
		replayed = makeReorder(db.noms, commit.BasisRef(), commit.Meta.Reorder.Origin, commit.Meta.Reorder.Date, commit.Target(), newBundle, newData)

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
		return fmt.Errorf("Invalid commit %s: diff: %s", commit.Original.Hash(), diff.Diff(commit.Original, replayed.Original))
	}

	return nil
}

func commonAncestor(r1, r2 types.Ref, noms types.ValueReader) (a types.Ref, err error) {
	fp, ok := datas.FindCommonAncestor(r1, r2, noms)
	if !ok {
		return a, fmt.Errorf("No common ancestor between commits: %s and %s", r1.TargetHash(), r2.TargetHash())
	}
	return fp, nil
}
