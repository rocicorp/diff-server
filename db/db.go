package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/exec"
	"github.com/aboodman/replicant/util/time"
)

const (
	local_dataset  = "local"
	remote_dataset = "remote"
)

type DB struct {
	noms   datas.Database
	origin string
	head   Commit
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
	err := r.init()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (db *DB) init() error {
	ds := db.noms.GetDataset(local_dataset)
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
	_, err := db.execInternal(types.Blob{}, ".putValue", types.NewList(db.noms, types.String(path), v))
	return err
}

func (db *DB) Del(path string) (ok bool, err error) {
	v, err := db.execInternal(types.Blob{}, ".delValue", types.NewList(db.noms, types.String(path)))
	return bool(v.(types.Bool)), err
}

func (db *DB) Bundle() (types.Blob, error) {
	return db.head.Bundle(db.noms), nil
}

func (db *DB) PutBundle(b types.Blob) error {
	_, err := db.execInternal(types.Blob{}, ".putBundle", types.NewList(db.noms, b))
	return err
}

func (db *DB) Exec(function string, args types.List) (types.Value, error) {
	if strings.HasPrefix(function, ".") {
		return nil, fmt.Errorf("Cannot call system function: %s", function)
	}
	return db.execInternal(db.head.Bundle(db.noms), function, args)
}

func (db *DB) Reload() error {
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
	_, err = db.noms.FastForward(db.noms.GetDataset(local_dataset), commitRef)
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
			newBundle = db.noms.WriteValue(args.Get(uint64(0)).(types.Blob))
			isWrite = true
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
