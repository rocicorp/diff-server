package db

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"roci.dev/replicant/util/noms/jsonpatch"
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

	if !fc.Value.Code.Equals(db.head.Value.Code) {
		buf, err := ioutil.ReadAll(db.head.Bundle(db.Noms()).Reader())
		if err != nil {
			return nil, err
		}
		j, err := json.Marshal(string(buf))
		if err != nil {
			return nil, err
		}
		r = append(r, jsonpatch.Operation{
			Op:    jsonpatch.OpReplace,
			Path:  "/s/code",
			Value: json.RawMessage(j),
		})
	}

	return r, nil
}
