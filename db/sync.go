package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

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

	// 3: Pull remote head to client
	datas.Pull(remoteDB.noms, db.noms, types.NewRef(rebased.Original), progress)

	// 4: Save the new remote state - primarily to avoid re-downloading it in the future and for debugging purposes.
	_, err = db.noms.SetHead(db.noms.GetDataset(remote_dataset), types.NewRef(rebased.Original))
	if err != nil {
		return err
	}

	// 5: Rebase any new local changes from between 1 and 3.
	reachable := reachable.New(db.noms)
	reachable.Populate(db.head.Original.Hash())
	rebased, err = rebase(db, reachable, types.NewRef(rebased.Original), datetime.Now(), db.head)
	if err != nil {
		return err
	}

	// 6: Commit new local head.
	_, err = db.noms.FastForward(db.noms.GetDataset(local_dataset), db.noms.WriteValue(rebased.Original))
	if err != nil {
		return err
	}

	return db.load()
}

func (db *DB) handleSync(commit Commit) (newHead Commit, err error) {
	// TODO: This needs to be done differently.
	// See: https://github.com/aboodman/replicant/issues/11
	reachable := reachable.New(db.noms)
	reachable.Populate(db.head.Original.Hash())

	err = validate(db, reachable, commit)
	if err != nil {
		return Commit{}, err
	}
	rebased, err := rebase(db, reachable, types.NewRef(db.head.Original), datetime.Now(), commit)
	if err != nil {
		return Commit{}, err
	}
	_, err = db.noms.FastForward(db.noms.GetDataset(local_dataset), db.noms.WriteValue(rebased.Original))
	if err != nil {
		return Commit{}, err
	}
	err = db.load()
	if err != nil {
		return Commit{}, err
	}
	return rebased, nil
}
