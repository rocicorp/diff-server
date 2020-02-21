package db

import (
	"errors"
	"log"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"roci.dev/diff-server/util/noms/jsonpatch"
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
		fc = makeCommit(db.Noms(), types.Ref{}, datetime.Epoch, db.noms.WriteValue(types.NewMap(db.noms)))
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
