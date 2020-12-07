// Package account is a lightweight account system for Replicache customers.
package account

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
)

type DB struct {
	ds datas.Dataset

	mu   sync.Mutex
	head Commit
}

type Commit struct {
	Parents []types.Ref `noms:",set"`
	Meta    struct {
	}
	Value Accounts
	//NomsStruct types.Struct `noms:",original"`
}

// Account records.
type Accounts struct {
	NextASID   uint32
	AutoSignup map[uint32]ASAccount
}

// An individual AutoSignup account record.
type ASAccount struct {
	ASID  uint32
	Name  string
	Email string
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

const LowestASID uint32 = 1000000

func (db *DB) initLocked() error {
	if !db.ds.HasHead() {
		fmt.Printf("no head\n")
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

func (db *DB) HeadValue() Accounts {
	defer db.lock()()
	return db.head.Value
}

// setHeadWithValue creates a new Commit with accounts as its value and sets head to it.
// If setHead returns a RetryError, reload head, apply changes, and try again.
func (db *DB) setHeadWithValue(accounts Accounts) error {
	defer db.lock()()
	return db.setHeadLocked(Commit{Value: accounts})
}

func (db *DB) setHeadLocked(newHead Commit) error {
	v, err := marshal.Marshal(db.Noms(), newHead)
	if err != nil {
		return err
	}
	ref := db.Noms().WriteValue(v)
	if _, err = db.Noms().SetHead(db.ds, ref); err != nil {
		return RetryError{err}
	}
	fmt.Printf("success, hashead: %v\n", db.ds.HasHead())
	db.head = newHead
	return nil
}

// RetryError indicates someone set head out from under us and the operation
// should be retried (re-load the new head, apply the changes, and attempt to
// set head again).
type RetryError struct {
	wrapped error
}

func (e RetryError) Error() string {
	return fmt.Sprintf("RetryError: %v", e.wrapped)
}

// func (db *DB) Hash() hash.Hash {
// return db.Head().NomsStruct.Hash()
// }

func (db *DB) Reload() error {
	db.lock()()
	fmt.Printf("has head %v\n", db.ds.HasHead())
	db.ds.Database().Rebase()
	fmt.Printf("has head %v\n", db.ds.HasHead())
	return db.initLocked()
}

// // Read reads the Commit with the given hash from the db.
// func Read(noms types.ValueReadWriter, hash hash.Hash) (Commit, error) {
// if hash.IsEmpty() {
// return Commit{}, errors.New("commit (empty hash) not found")
// }
// v := noms.ReadValue(hash)
// if v == nil {
// return Commit{}, fmt.Errorf("commit %s not found", hash)
// }
// var c Commit
// err := marshal.Unmarshal(v, &c)
// return c, err
// }

// // MaybePutData creates a new commit with the given map and lastMutationID if
// // they are different from what is currently at head. It returns the new Commit
// // if written or a zero value Commit if not (commit.NomsStruct.IsZeroValue() will be true).
// func (db *DB) MaybePutData(m kv.Map, lastMutationID uint64) (Commit, error) {
// defer db.lock()()

// hv := db.head.Value
// hvc, err := kv.ChecksumFromString(string(hv.Checksum))
// if err != nil {
// return Commit{}, fmt.Errorf("couldnt parse checksum from commit: %w", err)
// }
// if lastMutationID == uint64(hv.LastMutationID) && m.Checksum() == hvc.String() {
// return Commit{}, nil
// }
// basis := types.NewRef(db.head.NomsStruct)
// commit := makeCommit(db.Noms(), basis, time.DateTime(), db.Noms().WriteValue(m.NomsMap()), m.NomsChecksum(), lastMutationID)
// db.Noms().WriteValue(commit.NomsStruct)
// if err := db.setHeadLocked(commit); err != nil {
// return Commit{}, err
// }
// return commit, nil
// }

func (db *DB) lock() func() {
	db.mu.Lock()
	return func() {
		db.mu.Unlock()
	}
}
