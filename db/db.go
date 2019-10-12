// Package db implements the core database abstraction of Replicant. It provides facilities to import
// transaction bundles, execute transactions, and synchronize Replicant databases.
package db

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/exec"
	"github.com/aboodman/replicant/util/time"
)

const (
	LOCAL_DATASET  = "local"
	REMOTE_DATASET = "remote"
)

type DB struct {
	noms   datas.Database
	origin string
	head   Commit
	mu     sync.Mutex
}

func Load(sp spec.Spec, origin string) (*DB, error) {
	if !sp.Path.IsEmpty() {
		return nil, errors.New("Invalid spec - must not specify a path")
	}

	return New(sp.GetDatabase(), origin)
}

func New(noms datas.Database, origin string) (*DB, error) {
	r := DB{
		noms:   noms,
		origin: origin,
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
		genesis := makeGenesis(db.noms)
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
		return makeGenesis(db.noms), nil
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

func (db *DB) Bundle() (types.Blob, error) {
	return db.head.Bundle(db.noms), nil
}

func (db *DB) PutBundle(b types.Blob) error {
	defer db.lock()()
	_, err := db.execInternal(types.Blob{}, ".putBundle", types.NewList(db.noms, b))
	return err
}

func (db *DB) Exec(function string, args types.List) (types.Value, error) {
	defer db.lock()()
	if strings.HasPrefix(function, ".") {
		return nil, fmt.Errorf("Cannot call system function: %s", function)
	}
	return db.execInternal(db.head.Bundle(db.noms), function, args)
}

func (db *DB) Reload() error {
	defer db.lock()()
	db.noms.Rebase()
	return db.init()
}

func (db *DB) execInternal(bundle types.Blob, function string, args types.List) (types.Value, error) {
	basis := types.NewRef(db.head.Original)
	newBundle, newData, output, isWrite, err := db.execImpl(basis, bundle, function, args)
	if err != nil {
		return nil, err
	}

	// Do not add commits for read-only transactions.
	if !isWrite {
		return output, nil
	}

	var bundleRef types.Ref
	if bundle != (types.Blob{}) {
		bundleRef = types.NewRef(bundle)
	}

	commit := makeTx(db.noms, basis, db.origin, time.DateTime(), bundleRef, function, args, newBundle, newData)
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
func (db *DB) execImpl(basis types.Ref, bundle types.Blob, function string, args types.List) (newBundleRef types.Ref, newDataRef types.Ref, output types.Value, isWrite bool, err error) {
	var basisCommit Commit
	err = marshal.Unmarshal(basis.TargetValue(db.noms), &basisCommit)
	if err != nil {
		return types.Ref{}, types.Ref{}, nil, false, err
	}

	newBundle := basisCommit.Value.Code
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
		case ".putBundle":
			cb := basisCommit.Bundle(db.noms)
			nb := args.Get(uint64(0)).(types.Blob)
			if cb.Equals(nb) {
				log.Printf("Bundle %s already installed, skipping", cb.Hash())
				break
			}

			var currentVersion, newVersion float64
			currentVersion, err = getBundleVersion(cb, db.noms)
			if err != nil {
				return
			}
			newVersion, err = getBundleVersion(nb, db.noms)
			if err != nil {
				return
			}
			shouldUpdate := func() bool {
				if currentVersion == 0 && newVersion == 0 {
					log.Printf("Replacing unversioned bundle %s with %s", cb.Hash(), nb.Hash())
					return true
				}
				if newVersion > currentVersion {
					log.Printf("Upgrading bundle from %f to %f", currentVersion, newVersion)
					return true
				}
				log.Printf("Proposed bundle version %f not better than current version %f, skipping update", newVersion, currentVersion)
				return false
			}
			if shouldUpdate() {
				newBundle = db.noms.WriteValue(nb)
				isWrite = true
			}
			break
		}
	} else {
		ed := &editor{noms: db.noms, data: basisCommit.Data(db.noms).Edit()}
		o, err := exec.Run(ed, bundle.Reader(), function, args)
		if err != nil {
			return types.Ref{}, types.Ref{}, nil, false, err
		}
		isWrite = ed.receivedMutAttempt
		newData = db.noms.WriteValue(ed.Finalize())
		output = o
	}

	return newBundle, newData, output, isWrite, nil
}

func (db *DB) lock() func() {
	db.mu.Lock()
	return func() {
		db.mu.Unlock()
	}
}

func getBundleVersion(bundle types.Blob, noms types.ValueReadWriter) (float64, error) {
	// TODO: Passing editor is not really necessary because the script cannot call
	// it because it only has access to the `db` object which is passed as a param
	// to transaction functions. We only need to pass something here because the
	// impl of exec.Run() requires ed.noms.
	ed := &editor{noms: noms, data: nil}
	r, err := exec.Run(ed, bundle.Reader(), "codeVersion", types.NewList(noms))
	if _, ok := err.(exec.UnknownFunctionError); ok {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if r == nil || r.Kind() != types.NumberKind {
		return 0, errors.New("codeVersion() must return a number")
	}
	return float64(r.(types.Number)), nil
}
