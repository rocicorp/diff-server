package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/exec"
)

const (
	local_dataset  = "local"
	remote_dataset = "remote"
)

var (
	schema = nomdl.MustParseType(`
Struct Commit {
	parents: Set<Ref<Cycle<Commit>>>,
	meta: Struct {
		origin: String,
		date: Struct DateTime {
			secSinceEpoch: Number,
		},
		op?: Struct Tx {  // op omitted for genesis commit
			code?: Ref<Blob>,  // code omitted for system functions
			name: String,
			args: List<Value>,
		} |
		Struct Reorder {
			Target: Ref<Cycle<Commit>>,
		} |
		Struct Reject {
			Target: Ref<Cycle<Commit>>,
			Detail: Value
		},
	},
	value: Struct {
		code: Ref<Blob>,
		data: Ref<Map<String, Value>>,
	},
}`)
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
	ds := noms.GetDataset(local_dataset)

	r := DB{
		noms:   noms,
		origin: origin,
	}

	if !ds.HasHead() {
		genesis := makeGenesis(noms)
		genRef := noms.WriteValue(genesis.Original)
		_, err := noms.FastForward(ds, genRef)
		if err != nil {
			return nil, err
		}
		r.head = genesis
	} else {
		headType := types.TypeOf(ds.Head())
		if !types.IsSubtype(schema, headType) {
			return nil, fmt.Errorf("Cannot load database. Specified head has non-Replicant data of type: %s", headType.Describe())
		}

		var head Commit
		err := marshal.Unmarshal(ds.Head(), &head)
		if err != nil {
			return nil, err
		}

		r.head = head
	}

	return &r, nil
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

func (db *DB) execInternal(bundle types.Blob, function string, args types.List) (error) {
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
	oldBundle := bundle
	newBundle := bundle
	oldData := db.head.Data(db.noms)
	ed := editor{db.noms, oldData.Edit()}

	if strings.HasPrefix(function, ".") {
		switch function {
		case ".putValue":
			k := args.Get(uint64(0))
			v := args.Get(uint64(1))
			ed.Put(string(k.(types.String)), v)
			break
		case ".putBundle":
			newBundle = args.Get(uint64(0)).(types.Blob)
			break
		}
	} else {
		err := exec.Run(ed, oldBundle.Reader(), function, args)
		if err != nil {
			return types.Ref{}, types.Ref{}, err
		}
	}

	newData := ed.Finalize()

	return db.noms.WriteValue(newBundle), db.noms.WriteValue(newData), nil
}
