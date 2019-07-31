package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/exec"
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

	noms := sp.GetDatabase()
	r := DB{
		noms:   noms,
		origin: origin,
	}
	err := r.load()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (db *DB) load() error {
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

func (db *DB) Has(id string) (bool, error) {
	return db.head.Data(db.noms).Has(types.String(id)), nil
}

func (db *DB) Get(id string) (types.Value, error) {
	return db.head.Data(db.noms).Get(types.String(id)), nil
}

func (db *DB) Put(path string, v types.Value) error {
	return db.execInternal(types.Blob{}, ".putValue", types.NewList(db.noms, types.String(path), v))
}

func (db *DB) Bundle() (types.Blob, error) {
	return db.head.Bundle(db.noms), nil
}

func (db *DB) PutBundle(b types.Blob) error {
	return db.execInternal(types.Blob{}, ".putBundle", types.NewList(db.noms, b))
}

func (db *DB) Exec(function string, args types.List) error {
	if strings.HasPrefix(function, ".") {
		return fmt.Errorf("Cannot call system function: %s", function)
	}
	return db.execInternal(db.head.Bundle(db.noms), function, args)
}

func (db *DB) Reload() error {
	db.noms.Rebase()
	return db.load()
}

func (db *DB) execInternal(bundle types.Blob, function string, args types.List) error {
	basis := types.NewRef(db.head.Original)
	newBundle, newData, err := db.execImpl(basis, bundle, function, args)
	if err != nil {
		return err
	}

	var bundleRef types.Ref
	if bundle != (types.Blob{}) {
		bundleRef = types.NewRef(bundle)
	}

	commit := makeTx(db.noms, basis, db.origin, datetime.Now(), bundleRef, function, args, newBundle, newData)
	commitRef := db.noms.WriteValue(commit.Original)

	// FastForward not strictly needed here because we should have already ensured that we were
	// fast-forwarding outside of Noms, but it's a nice sanity check.
	_, err = db.noms.FastForward(db.noms.GetDataset(local_dataset), commitRef)
	if err != nil {
		return err
	}
	db.head = commit
	return nil
}

// TODO: add date and random source to this so that sync can set it up correctly when replaying.
func (db *DB) execImpl(basis types.Ref, bundle types.Blob, function string, args types.List) (newBundleRef types.Ref, newDataRef types.Ref, err error) {
	var basisCommit Commit
	err = marshal.Unmarshal(basis.TargetValue(db.noms), &basisCommit)
	if err != nil {
		return types.Ref{}, types.Ref{}, err
	}

	newBundle := basisCommit.Value.Code
	newData := basisCommit.Value.Data

	if strings.HasPrefix(function, ".") {
		switch function {
		case ".putValue":
			k := args.Get(uint64(0))
			v := args.Get(uint64(1))
			ed := editor{db.noms, basisCommit.Data(db.noms).Edit()}
			ed.Put(string(k.(types.String)), v)
			newData = db.noms.WriteValue(ed.Finalize())
			break
		case ".putBundle":
			newBundle = db.noms.WriteValue(args.Get(uint64(0)).(types.Blob))
			break
		}
	} else {
		ed := editor{db.noms, basisCommit.Data(db.noms).Edit()}
		err := exec.Run(ed, bundle.Reader(), function, args)
		if err != nil {
			return types.Ref{}, types.Ref{}, err
		}
		newData = db.noms.WriteValue(ed.Finalize())
	}

	return newBundle, newData, nil
}
