package db

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/util/time"
)

func (db *DB) Sync(remote spec.Spec) error {
	// No lock here because we're doing http requests.
	localHead := db.head

	remoteDB, err := Load(remote, fmt.Sprintf("%s/remote", db.origin))
	if err != nil {
		return err
	}

	progress := make(chan datas.PullProgress)
	go func() {
		for p := range progress {
			log.Println("pull progress", p)
		}
	}()

	// 1: Push client head to server
	datas.Pull(db.noms, remoteDB.noms, localHead.Ref(), progress)

	// 2: Merge client changes into server state
	var remoteHead types.Ref
	if remote.Protocol == "http" || remote.Protocol == "https" {
		remoteHead, err = remoteSync(remote, remoteDB, localHead)
	} else {
		remoteHead, err = HandleSync(remoteDB, localHead)
	}
	if err != nil {
		return err
	}

	// 3: Pull remote head to client
	datas.Pull(remoteDB.noms, db.noms, remoteHead, progress)

	// Lock here because all work from here on out is local and we are going to read/write
	// local state.
	defer db.lock()()
	localHead = db.head

	// 4: Save the new remote state - primarily to avoid re-downloading it in the future and for debugging purposes.
	_, err = db.noms.SetHead(db.noms.GetDataset(REMOTE_DATASET), remoteHead)
	if err != nil {
		return err
	}

	// 5: Rebase any new local changes from between 1 and 3.
	rebased, err := rebase(db, remoteHead, time.DateTime(), localHead, types.Ref{})
	if err != nil {
		return err
	}

	// 6: Commit new local head.
	_, err = db.noms.FastForward(db.noms.GetDataset(LOCAL_DATASET), db.noms.WriteValue(rebased.Original))
	if err != nil {
		return err
	}

	return db.init()
}

func remoteSync(remote spec.Spec, remoteDB *DB, commit Commit) (newHead types.Ref, err error) {
	url := remote.String() + "/sync?head=" + commit.Original.Hash().String()
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return newHead, err
	}
	req.Header.Set(datas.NomsVersionHeader, constants.NomsVersion)
	req.Header.Set("Authorization", remote.Options.Authorization)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return newHead, err
	}
	if resp.StatusCode != http.StatusOK {
		io.Copy(os.Stderr, resp.Body)
		return newHead, fmt.Errorf("Sync to %s failed with: %d", url, resp.StatusCode)
	}
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return newHead, err
	}
	h, ok := hash.MaybeParse(buf.String())
	if !ok {
		return newHead, errors.New("Could not parse sync response from server as hash: " + buf.String())
	}
	v := remoteDB.Noms().ReadValue(h)
	if v == nil {
		return newHead, fmt.Errorf("Could not read merged head '%s' from remote server", h.String())
	}
	return types.NewRef(v), nil
}

// HandleSync implements the server-side of the sync protocol. It's not typical to call it
// directly, and is exposed primarily so that the server implementation can call it.
func HandleSync(dest *DB, commit Commit) (newHead types.Ref, err error) {
	rebased, err := rebase(dest, types.NewRef(dest.head.Original), time.DateTime(), commit, types.Ref{})
	if err != nil {
		return newHead, err
	}
	_, err = dest.noms.FastForward(dest.noms.GetDataset(LOCAL_DATASET), dest.noms.WriteValue(rebased.Original))
	if err != nil {
		return newHead, err
	}
	err = dest.init()
	if err != nil {
		return newHead, err
	}
	return types.NewRef(rebased.Original), nil
}
