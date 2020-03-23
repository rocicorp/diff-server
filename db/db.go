// Package db implements the core database abstraction of Replicant. It provides facilities to import
// transaction bundles, execute transactions, and synchronize Replicant databases.
package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/time"
)

type DB struct {
	ds   datas.Dataset
	head Commit
}

func New(ds datas.Dataset) (*DB, error) {
	r := DB{
		ds: ds,
	}
	err := r.init()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (db *DB) init() error {
	if !db.ds.HasHead() {
		m := kv.NewMap(db.Noms())
		genesis := makeCommit(db.Noms(),
			types.Ref{},
			time.DateTime(),
			db.Noms().WriteValue(m.NomsMap()),
			types.String(m.Checksum().String()),
			"" /*lastMutationID*/)
		genRef := db.Noms().WriteValue(genesis.Original)
		_, err := db.ds.Database().FastForward(db.ds, genRef)
		if err != nil {
			return err
		}
		db.head = genesis
		return nil
	}

	headType := types.TypeOf(db.ds.Head())
	if !types.IsSubtype(schema, headType) {
		return fmt.Errorf("Cannot load database. Specified head has non-Replicache data of type: %s", headType.Describe())
	}

	var head Commit
	err := marshal.Unmarshal(db.ds.Head(), &head)
	if err != nil {
		return err
	}

	db.head = head
	return nil
}

func (db *DB) Noms() datas.Database {
	return db.ds.Database()
}

func (db *DB) Head() Commit {
	return db.head
}

func (db *DB) Hash() hash.Hash {
	return db.head.Original.Hash()
}

func (db *DB) Reload() error {
	db.ds.Database().Rebase()
	return db.init()
}

func (db *DB) PutData(newData types.Map, checksum types.String, lastMutationID string) error {
	basis := types.NewRef(db.head.Original)
	commit := makeCommit(db.Noms(), basis, time.DateTime(), db.Noms().WriteValue(newData), checksum, lastMutationID)
	commitRef := db.Noms().WriteValue(commit.Original)

	// FastForward not strictly needed here because we should have already ensured that we were
	// fast-forwarding outside of Noms, but it's a nice sanity check.
	_, err := db.ds.Database().FastForward(db.ds, commitRef)
	if err != nil {
		return err
	}
	db.head = commit
	return nil
}
