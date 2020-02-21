// Package db implements the core database abstraction of Replicant. It provides facilities to import
// transaction bundles, execute transactions, and synchronize Replicant databases.
package db

import (
	"errors"
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"roci.dev/diff-server/util/time"
)

const (
	LOCAL_DATASET = "local"
)

type DB struct {
	noms datas.Database
	head Commit
	mu   sync.Mutex
}

func Load(sp spec.Spec) (*DB, error) {
	if !sp.Path.IsEmpty() {
		return nil, errors.New("Invalid spec - must not specify a path")
	}

	var noms datas.Database
	err := d.Try(func() {
		noms = sp.GetDatabase()
	})
	if err != nil {
		err = err.(d.WrappedError).Cause()
		return nil, err
	}
	return New(noms)
}

func New(noms datas.Database) (*DB, error) {
	r := DB{
		noms: noms,
	}
	defer r.lock()()
	err := r.init()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (db *DB) init() error {
	ds := db.noms.GetDataset(LOCAL_DATASET)
	if !ds.HasHead() {
		genesis := makeCommit(db.noms, types.Ref{}, datetime.Now(), db.Noms().WriteValue(types.NewMap(db.Noms())))
		genRef := db.noms.WriteValue(genesis.Original)
		_, err := db.noms.FastForward(ds, genRef)
		if err != nil {
			return err
		}
		db.head = genesis
		return nil
	}

	headType := types.TypeOf(ds.Head())
	if !types.IsSubtype(schema, headType) {
		return fmt.Errorf("Cannot load database. Specified head has non-Replicache data of type: %s", headType.Describe())
	}

	var head Commit
	err := marshal.Unmarshal(ds.Head(), &head)
	if err != nil {
		return err
	}

	db.head = head
	return nil
}

func (db *DB) Noms() types.ValueReadWriter {
	return db.noms
}

func (db *DB) Head() Commit {
	return db.head
}

func (db *DB) Hash() hash.Hash {
	return db.head.Original.Hash()
}

func (db *DB) Reload() error {
	defer db.lock()()
	db.noms.Rebase()
	return db.init()
}

func (db *DB) lock() func() {
	db.mu.Lock()
	return func() {
		db.mu.Unlock()
	}
}

func (db *DB) PutData(newData types.Map) error {
	defer db.lock()()
	basis := types.NewRef(db.head.Original)
	commit := makeCommit(db.noms, basis, time.DateTime(), db.noms.WriteValue(newData))
	commitRef := db.noms.WriteValue(commit.Original)

	// FastForward not strictly needed here because we should have already ensured that we were
	// fast-forwarding outside of Noms, but it's a nice sanity check.
	_, err := db.noms.FastForward(db.noms.GetDataset(LOCAL_DATASET), commitRef)
	if err != nil {
		return err
	}
	db.head = commit
	return nil
}
