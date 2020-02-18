// Package db implements the core database abstraction of Replicant. It provides facilities to import
// transaction bundles, execute transactions, and synchronize Replicant databases.
package db

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"

	"roci.dev/replicant/exec"
	"roci.dev/replicant/util/time"
)

const (
	LOCAL_DATASET  = "local"
	REMOTE_DATASET = "remote"
)

type DB struct {
	noms   datas.Database
	head   Commit
	bundle []byte
	mu     sync.Mutex
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
		genesis := makeGenesis(db.noms, "")
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
		return fmt.Errorf("Cannot load database. Specified head has non-Replicant data of type: %s", headType.Describe())
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

func (db *DB) RemoteHead() (c Commit, err error) {
	ds := db.noms.GetDataset(REMOTE_DATASET)
	if !ds.HasHead() {
		// TODO: maybe setup the remote head at startup too.
		return makeGenesis(db.noms, ""), nil
	}
	err = marshal.Unmarshal(ds.Head(), &c)
	return
}

func (db *DB) Hash() hash.Hash {
	return db.head.Original.Hash()
}

func (db *DB) Has(id string) (bool, error) {
	return db.head.Data(db.noms).Has(types.String(id)), nil
}

func (db *DB) Get(id string) (types.Value, error) {
	return db.head.Data(db.noms).Get(types.String(id)), nil
}

func (db *DB) Put(path string, v types.Value) error {
	defer db.lock()()
	_, err := db.execInternal(types.Blob{}, ".putValue", types.NewList(db.noms, types.String(path), v))
	return err
}

func (db *DB) Del(path string) (ok bool, err error) {
	defer db.lock()()
	v, err := db.execInternal(types.Blob{}, ".delValue", types.NewList(db.noms, types.String(path)))
	return bool(v.(types.Bool)), err
}

func (db *DB) Bundle() []byte {
	return db.bundle
}

func (db *DB) PutBundle(b []byte) error {
	err := validateBundle(b, db.noms)
	if err != nil {
		return err
	}
	defer db.lock()()
	db.bundle = b
	return nil
}

func (db *DB) Exec(function string, args types.List) (types.Value, error) {
	r, be, err := db.ExecBatch([]BatchItem{
		BatchItem{
			Function: function,
			Args:     args,
		},
	})
	if err != nil {
		return nil, err
	}
	if be != nil {
		return nil, be.error
	}
	return r[0].Result, nil
}

type BatchItem struct {
	Function string
	Args     types.List
}

type BatchItemResponse struct {
	Result types.Value
}

type BatchError struct {
	error
	Index int
}

// ExecBatch executes zero  or more transactions against the database atomically.
// If a transaction fails, the returned error will be BatchError with the corresponding
// index set correctly.
func (db *DB) ExecBatch(batch []BatchItem) ([]BatchItemResponse, *BatchError, error) {
	defer db.lock()()
	ds := db.noms.GetDataset(LOCAL_DATASET)
	oldHead := ds.HeadRef()
	r := make([]BatchItemResponse, 0, len(batch))
	basis := db.head
	basisRef := basis.Ref()
	for i, item := range batch {
		if strings.HasPrefix(item.Function, ".") {
			return nil, &BatchError{fmt.Errorf("Cannot call system function: %s", item.Function), i}, nil
		}
		newData, output, isWrite, err := db.execImpl(basisRef, item.Function, item.Args)
		if err != nil {
			return nil, &BatchError{err, i}, nil
		}

		r = append(r, BatchItemResponse{})
		itemRes := &r[len(r)-1]
		itemRes.Result = output

		// Do not add commits for read-only transactions.
		if !isWrite {
			continue
		}

		basis = makeTx(db.noms, basisRef, time.DateTime(), item.Function, item.Args, newData)
		basisRef = db.noms.WriteValue(basis.Original)
	}

	// FastForward not strictly needed here because we should have already ensured that we were
	// fast-forwarding outside of Noms, but it's a nice sanity check.
	newDS, err := db.noms.FastForward(ds, basisRef)
	if err != nil {
		db.noms.Flush()
		log.Printf("Error committing execBatch - error: %s, old head: %s, attempted head: %s, current head: %s", err, oldHead.TargetHash(), basisRef.TargetHash(), newDS.Head().Hash())
		return r, nil, err
	}
	db.head = basis
	return r, nil, nil
}

func (db *DB) Reload() error {
	defer db.lock()()
	db.noms.Rebase()
	return db.init()
}

func (db *DB) execInternal(bundle types.Blob, function string, args types.List) (types.Value, error) {
	basis := types.NewRef(db.head.Original)
	newData, output, isWrite, err := db.execImpl(basis, function, args)
	if err != nil {
		return nil, err
	}

	// Do not add commits for read-only transactions.
	if !isWrite {
		return output, nil
	}

	commit := makeTx(db.noms, basis, time.DateTime(), function, args, newData)
	commitRef := db.noms.WriteValue(commit.Original)

	// FastForward not strictly needed here because we should have already ensured that we were
	// fast-forwarding outside of Noms, but it's a nice sanity check.
	_, err = db.noms.FastForward(db.noms.GetDataset(LOCAL_DATASET), commitRef)
	if err != nil {
		return nil, err
	}
	db.head = commit
	return output, nil
}

// TODO: add date and random source to this so that sync can set it up correctly when replaying.
func (db *DB) execImpl(basis types.Ref, function string, args types.List) (newDataRef types.Ref, output types.Value, isWrite bool, err error) {
	var basisCommit Commit
	err = marshal.Unmarshal(basis.TargetValue(db.noms), &basisCommit)
	if err != nil {
		return types.Ref{}, nil, false, err
	}

	newData := basisCommit.Value.Data

	if strings.HasPrefix(function, ".") {
		switch function {
		case ".putValue":
			k := args.Get(uint64(0))
			v := args.Get(uint64(1))
			ed := editor{noms: db.noms, data: basisCommit.Data(db.noms).Edit()}
			isWrite = true
			err = ed.Put(string(k.(types.String)), v)
			if err != nil {
				return
			}
			newData = db.noms.WriteValue(ed.Finalize())
			break
		case ".delValue":
			k := args.Get(uint64(0))
			ed := editor{noms: db.noms, data: basisCommit.Data(db.noms).Edit()}
			isWrite = true
			var ok bool
			ok, err = ed.Del(string(k.(types.String)))
			if err != nil {
				return
			}
			newData = db.noms.WriteValue(ed.Finalize())
			output = types.Bool(ok)
			break
		}
	} else {
		ed := &editor{noms: db.noms, data: basisCommit.Data(db.noms).Edit()}
		o, err := exec.Run(ed, bytes.NewReader(db.bundle), function, args)
		if err != nil {
			return types.Ref{}, nil, false, err
		}
		isWrite = ed.receivedMutAttempt
		newData = db.noms.WriteValue(ed.Finalize())
		output = o
	}

	return newData, output, isWrite, nil
}

func (db *DB) lock() func() {
	db.mu.Lock()
	return func() {
		db.mu.Unlock()
	}
}

func validateBundle(bundle []byte, noms types.ValueReadWriter) error {
	// TODO: Passing editor is not really necessary because the script cannot call
	// it because it only has access to the `db` object which is passed as a param
	// to transaction functions. We only need to pass something here because the
	// impl of exec.Run() requires ed.noms.
	ed := &editor{noms: noms, data: nil}
	_, err := exec.Run(ed, bytes.NewReader(bundle), "", types.NewList(noms))
	return err
}
