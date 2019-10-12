package db

import "fmt"

// Replays the provided transaction against its basis and returns the resulting commit.
// Callers can determine if a commit is valid by comparing the replayed version to the original.
func validate(db *DB, commit Commit) (replayed Commit, err error) {
	switch commit.Type() {
	case CommitTypeTx:
		newBundle, newData, _, _, err := db.execImpl(commit.BasisRef(), commit.Meta.Tx.Bundle(db.noms), commit.Meta.Tx.Name, commit.Meta.Tx.Args)
		if err != nil {
			return Commit{}, err
		}
		return makeTx(db.noms, commit.BasisRef(), commit.Meta.Tx.Origin, commit.Meta.Tx.Date, commit.Meta.Tx.Code, commit.Meta.Tx.Name, commit.Meta.Tx.Args, newBundle, newData), nil

	case CommitTypeReorder:
		target, err := commit.InitalCommit(db.noms)
		if err != nil {
			return Commit{}, err
		}
		newBundle, newData, _, _, err := db.execImpl(commit.BasisRef(), target.Meta.Tx.Bundle(db.noms), target.Meta.Tx.Name, target.Meta.Tx.Args)
		if err != nil {
			return Commit{}, err
		}
		return makeReorder(db.noms, commit.BasisRef(), commit.Meta.Reorder.Origin, commit.Meta.Reorder.Date, commit.Target(), newBundle, newData), nil
	}

	// We should never get asked to validate other commit types:
	// - reject: clients aren't allowed to create reject commits, only the server, so forks should never contain rejects
	// - genesis: the genesis commit should never be in a fork

	return Commit{}, fmt.Errorf("Invalid commit type: %v", commit.Type())
}
