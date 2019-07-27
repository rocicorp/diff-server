package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/util/history"
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
	localHead := db.head
	rebased, err := remoteDB.handleSync(localHead)
	if err != nil {
		return err
	}

	_, err = db.noms.SetHead(db.noms.GetDataset(remote_dataset), types.NewRef(rebased.Original))
	if err != nil {
		return err
	}

	rebased, err = rebase(db, types.NewRef(rebased.Original), db.head)
	if err != nil {
		return err
	}

	_, err = db.noms.FastForward(db.noms.GetDataset(local_dataset), db.noms.WriteValue(rebased.Original))
	return err
}

func (db *DB) handleSync(commit Commit) (newHead Commit, err error) {
	cache := history.NewCache(db.noms)
	err = validate(db, cache, commit)
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
