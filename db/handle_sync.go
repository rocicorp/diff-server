package db

import (
	"errors"
	"log"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"roci.dev/replicant/util/noms/jsonpatch"
	"roci.dev/replicant/util/time"
)

func (db *DB) HandleSync(from hash.Hash) ([]jsonpatch.Operation, error) {
	if from == db.head.Original.Hash() {
		return []jsonpatch.Operation{}, nil
	}

	r := []jsonpatch.Operation{}
	v := db.Noms().ReadValue(from)
	var fc Commit
	var err error
	if v == nil {
		log.Printf("Error: Requested sync basis %s could not be found - sending a fresh sync", from)
		r = append(r, jsonpatch.Operation{
			Op:   jsonpatch.OpRemove,
			Path: "/",
		})
		fc = makeGenesis(db.Noms(), "")
	} else {
		err = marshal.Unmarshal(v, &fc)
		if err != nil {
			log.Printf("Error: Requested sync basis %s is not a commit: %#v", from, v)
			return nil, errors.New("Invalid commitID")
		}
	}

	if !fc.Value.Data.Equals(db.head.Value.Data) {
		fm := fc.Data(db.Noms())
		i := len(r)
		r, err = jsonpatch.Diff(fm, db.head.Data(db.Noms()), r)
		if err != nil {
			return nil, err
		}
		for ; i < len(r); i++ {
			r[i].Path = "/u" + r[i].Path
		}
	}

	return r, nil
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
