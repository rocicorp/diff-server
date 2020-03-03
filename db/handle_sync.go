package db

import (
	"errors"
	"log"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"roci.dev/diff-server/kv"
)

func (db *DB) HandleSync(from hash.Hash) ([]kv.Operation, error) {
	if from == db.head.Original.Hash() {
		return []kv.Operation{}, nil
	}

	r := []kv.Operation{}
	v := db.Noms().ReadValue(from)
	var fc Commit
	var err error
	if v == nil {
		log.Printf("Requested sync basis %s could not be found - sending a fresh sync", from)
		r = append(r, kv.Operation{
			Op:   kv.OpRemove,
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
		fm := kv.NewMapFromNoms(db.Noms(), fc.Data(db.Noms()))
		tm := kv.NewMapFromNoms(db.Noms(), db.head.Data(db.Noms()))
		r, err = kv.Diff(fm, tm, r)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}
