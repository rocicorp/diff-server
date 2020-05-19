// Package db implements the core database abstraction of Replicant. It provides facilities to import
// transaction bundles, execute transactions, and synchronize Replicant databases.
package db

import (
	"errors"
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/time"
)

type DB struct {
	ds datas.Dataset

	mu   sync.Mutex
	head Commit
}

func New(ds datas.Dataset) (*DB, error) {
	r := DB{
		ds: ds,
	}
	defer r.lock()()
	err := r.initLocked()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (db *DB) initLocked() error {
	if !db.ds.HasHead() {
		m := kv.NewMap(db.Noms())
		genesis := makeCommit(db.Noms(),
			types.Ref{},
			time.DateTime(),
			db.Noms().WriteValue(m.NomsMap()),
			m.NomsChecksum(),
			0 /*lastMutationID*/)
		db.Noms().WriteValue(genesis.Original)
		return db.setHeadLocked(genesis)
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
	defer db.lock()()
	return db.head
}

// setHead sets the head commit to newHead and fast-forwards the underlying dataset.
func (db *DB) setHead(newHead Commit) error {
	defer db.lock()()
	return db.setHeadLocked(newHead)
}

func (db *DB) setHeadLocked(newHead Commit) error {
	_, err := db.Noms().FastForward(db.ds, newHead.Ref())
	if err != nil {
		return err
	}
	db.head = newHead
	return nil
}

func (db *DB) Hash() hash.Hash {
	return db.Head().Original.Hash()
}

func (db *DB) Reload() error {
	db.lock()()
	db.ds.Database().Rebase()
	return db.initLocked()
}

// Read reads the Commit with the given hash from the db.
func Read(noms types.ValueReadWriter, hash hash.Hash) (Commit, error) {
	if hash.IsEmpty() {
		return Commit{}, errors.New("commit (empty hash) not found")
	}
	v := noms.ReadValue(hash)
	if v == nil {
		return Commit{}, fmt.Errorf("commit %s not found", hash)
	}
	var c Commit
	err := marshal.Unmarshal(v, &c)
	return c, err
}

// MaybePutData creates a new commit with the given map and lastMutationID if
// they are different from what is currently at head. It returns the new Commit
// if written or a zero value Commit if not (commit.NomsStruct.IsZeroValue() will be true).
func (db *DB) MaybePutData(m kv.Map, lastMutationID uint64) (Commit, error) {
	defer db.lock()()

	hv := db.head.Value
	hvc, err := kv.ChecksumFromString(string(hv.Checksum))
	if err != nil {
		return Commit{}, fmt.Errorf("couldnt parse checksum from commit: %w", err)
	}
	if lastMutationID == uint64(hv.LastMutationID) && m.Checksum() == hvc.String() {
		return Commit{}, nil
	}
	basis := types.NewRef(db.head.Original)
	commit := makeCommit(db.Noms(), basis, time.DateTime(), db.Noms().WriteValue(m.NomsMap()), m.NomsChecksum(), lastMutationID)
	db.Noms().WriteValue(commit.Original)
	if err := db.setHeadLocked(commit); err != nil {
		return Commit{}, err
	}
	commit.Original.IsZeroValue()
	return commit, nil
}

func (db *DB) lock() func() {
	db.mu.Lock()
	return func() {
		db.mu.Unlock()
	}
}
