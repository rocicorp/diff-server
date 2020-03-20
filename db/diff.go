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

func fullSync(db *DB, from hash.Hash) ([]kv.Operation, Commit) {
	log.Printf("Requested sync basis %s could not be found - sending a full sync", from)
	r := []kv.Operation{
		kv.Operation{
			Op:   kv.OpRemove,
			Path: "/",
		},
	}
	m := kv.NewMap(db.noms)
	return r, makeCommit(db.Noms(), types.Ref{}, datetime.Epoch, db.noms.WriteValue(m.NomsMap()), types.String(m.Checksum().String()))
}

func (db *DB) Diff(from hash.Hash, fromChecksum kv.Checksum) ([]kv.Operation, error) {
	r := []kv.Operation{}
	v := db.Noms().ReadValue(from)
	var fc Commit
	var err error
	if v == nil {
		r, fc = fullSync(db, from)
	} else {
		err = marshal.Unmarshal(v, &fc)
		if err != nil {
			log.Printf("Error: Requested sync basis %s is not a commit: %#v", from, v)
			return nil, errors.New("Invalid baseStateID")
		}
	}

	fcChecksum, err := kv.ChecksumFromString(string(fc.Value.Checksum))
	if err != nil {
		log.Printf("Error: couldn't parse checksum from commit: %s", string(fc.Value.Checksum))
		return nil, errors.New("unable to parse commit checksum from db")
	} else if !fcChecksum.Equal(fromChecksum) {
		log.Printf("Error: checksum mismatch; %s from client, %s in db", fromChecksum.String(), fcChecksum.String())
		r, fc = fullSync(db, from)
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
