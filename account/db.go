// Package account is a lightweight account system for Replicache customers.
package account

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
)

// DB represents the Replicache account database. It is modeled on the pattern
// established by the main db (db/db.go).
type DB struct {
	ds datas.Dataset

	mu   sync.Mutex
	head Commit
}

// Commit is the Git-like commit structure Noms uses to store values.
// The account database keeps a single Commit at the head of its dataset
// containing all current entries.
type Commit struct {
	// Parents and Meta are unused.
	Parents []types.Ref `noms:",set"`
	Meta    struct {
	}

	Value Records
}

// NewDB returns a new account.DB. If we want the flexibility of using DB
// with multiple Noms databases or datasets we could break those out as
// parameters, but for now keeping it simpler.
func NewDB(storageRoot string) (*DB, error) {
	sp, err := spec.ForDatabase(fmt.Sprintf("%s/%s", storageRoot, DatabaseName))
	if err != nil {
		return nil, err
	}
	noms := sp.GetDatabase()
	ds := noms.GetDataset(DatasetName)
	r := DB{
		ds: ds,
	}
	defer r.lock()()
	err = r.initLocked()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// TODO change database to "accounts" once we release to reset our
//      tire-kicking before releasing the feature.
const DatabaseName = "TODOaccounts"
const DatasetName = "websignup"

func (db *DB) initLocked() error {
	if !db.ds.HasHead() {
		accounts := Records{
			NextASID: LowestASID,
			Record:   make(map[uint32]Record),
		}
		return db.setHeadLocked(Commit{Value: accounts})
	}

	var head Commit
	err := marshal.Unmarshal(db.ds.Head(), &head)
	if err != nil {
		return err
	}
	// Noms roundtrips empty maps as nil, so ensure we have a map.
	if head.Value.Record == nil {
		head.Value.Record = map[uint32]Record{}
	}

	db.head = head
	return nil
}

func (db *DB) Noms() datas.Database {
	return db.ds.Database()
}

// HeadValue returns the value at head. Note that the Records returned contains
// pointer types (eg, a map) so any changes to the pointer members of the Records
// returned will be visible to any other caller to whom this Records has been
// returned. Use CopyRecords to get a value that is safe to change. This is
// not a good pattern, especially because Noms might require retry on write,
// but luckily this is a temporary thing. (Reader two years in the future: <smirks>.)
func (db *DB) HeadValue() Records {
	defer db.lock()()
	return db.head.Value
}

// SetHeadWithValue creates a new Commit with accounts as its value and sets head to it.
// If setHead returns a RetryError, caller should reload head, re-apply changes, and
// try again (up to a few times).
func (db *DB) SetHeadWithValue(accounts Records) error {
	defer db.lock()()
	return db.setHeadLocked(Commit{Value: accounts})
}

func (db *DB) setHeadLocked(newHead Commit) error {
	v, err := marshal.Marshal(db.Noms(), newHead)
	if err != nil {
		return err
	}
	ref := db.Noms().WriteValue(v)
	var ds datas.Dataset
	if ds, err = db.Noms().SetHead(db.ds, ref); err != nil {
		// We could save the caller a reload of head in this case by returning
		// the new ds on error. It kinda complicates the caller tho, so didn't do it.
		return RetryError{err}
	}
	db.ds = ds
	db.head = newHead
	return nil
}

// RetryError indicates someone set head out from under us and the operation
// should be retried (re-load the new head, re-apply the changes, and attempt to
// set head again).
type RetryError struct {
	wrapped error
}

func (e RetryError) Error() string {
	return fmt.Sprintf("RetryError: %v", e.wrapped)
}

func (e *RetryError) Unwrap() error { return e.wrapped }

// Reload reloads the latest state from the underlying noms db.
func (db *DB) Reload() error {
	db.lock()()
	db.ds.Database().Rebase()
	db.ds = db.ds.Database().GetDataset(db.ds.ID())
	return db.initLocked()
}

func (db *DB) lock() func() {
	db.mu.Lock()
	return func() {
		db.mu.Unlock()
	}
}
