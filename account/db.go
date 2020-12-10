// Package account is a lightweight account system for Replicache customers.
package account

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
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

	Value Accounts
}

// Account records.
type Accounts struct {
	NextASID   uint32
	AutoSignup map[uint32]ASAccount // Map key is the ASID.
}

// An individual AutoSignup account record.
type ASAccount struct {
	ASID        uint32
	Name        string
	Email       string
	DateCreated string
}

func NewDB(ds datas.Dataset) (*DB, error) {
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

// ASIDs are issued in a separate range from regular accounts.
// See RFC: https://github.com/rocicorp/repc/issues/269
const LowestASID uint32 = 1000000

func (db *DB) initLocked() error {
	if !db.ds.HasHead() {
		accounts := Accounts{
			NextASID:   LowestASID,
			AutoSignup: make(map[uint32]ASAccount),
		}
		return db.setHeadLocked(Commit{Value: accounts})
	}

	var head Commit
	err := marshal.Unmarshal(db.ds.Head(), &head)
	if err != nil {
		return err
	}
	// Noms roundtrips empty maps as nil, so ensure we have a map.
	if head.Value.AutoSignup == nil {
		head.Value.AutoSignup = map[uint32]ASAccount{}
	}

	db.head = head
	return nil
}

func (db *DB) Noms() datas.Database {
	return db.ds.Database()
}

// Callers don't care about the underlying Commit structure, so we have
// HeadValue and SetHeadWithValue, unlike db/db.go, which has Head and SetHead.
func (db *DB) HeadValue() Accounts {
	defer db.lock()()
	return db.head.Value
}

// SetHeadWithValue creates a new Commit with accounts as its value and sets head to it.
// If setHead returns a RetryError, caller should reload head, re-apply changes, and
// try again (up to a few times).
func (db *DB) SetHeadWithValue(accounts Accounts) error {
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
