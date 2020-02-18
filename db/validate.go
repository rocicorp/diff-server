package db

import "fmt"

// Replays the provided transaction against its basis and returns the resulting commit.
// Callers can determine if a commit is valid by comparing the replayed version to the original.
func validate(db *DB, commit Commit) (replayed Commit, err error) {
	switch commit.Type() {
	case CommitTypeTx:
		newData, _, _, err := db.execImpl(commit.BasisRef(), commit.Meta.Tx.Name, commit.Meta.Tx.Args)
		if err != nil {
			return Commit{}, err
		}
		return makeTx(db.noms, commit.BasisRef(), commit.Meta.Tx.Date, commit.Meta.Tx.Name, commit.Meta.Tx.Args, newData), nil

	case CommitTypeReorder:
		target, err := commit.InitalCommit(db.noms)
		if err != nil {
			return Commit{}, err
		}
		newData, _, _, err := db.execImpl(commit.BasisRef(), target.Meta.Tx.Name, target.Meta.Tx.Args)
		if err != nil {
			return Commit{}, err
		}
		return makeReorder(db.noms, commit.BasisRef(), commit.Meta.Reorder.Date, commit.Target(), newData), nil
	}

	// We should never get asked to validate other commit types:
	// - genesis: the genesis commit should never be in a fork

	return Commit{}, fmt.Errorf("Invalid commit type: %v", commit.Type())
}
