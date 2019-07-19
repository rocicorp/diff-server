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
		date: Struct DateTime {
			secSinceEpoch: Number,
		},
		op?: Struct Tx {
			origin: String,
			code: Ref<Blob>,
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
		genesis := Commit{}
		genesis.Value.Data = noms.WriteValue(types.NewMap(noms))
		genesis.Value.Code = noms.WriteValue(types.NewBlob(noms))
		genesis.Original = marshal.MustMarshal(noms, genesis).(types.Struct)
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
	return db.execImpl(".putValue", types.NewList(db.noms, v))
}

func (db *DB) Bundle() (types.Blob, error) {
	return db.head.Bundle(db.noms), nil
}

func (db *DB) PutBundle(b types.Blob) error {
	return db.execImpl(".putBundle", types.NewList(db.noms, b))
}

func (db *DB) Exec(function string, args types.List) error {
	if strings.HasPrefix(function, ".") {
		return fmt.Errorf("Cannot call system function: %s", function)
	}
	return db.execImpl(function, args)
}

func (db *DB) Sync(remote spec.Spec) error {
	return nil
}

// This is the one that will get called during sync, or one like it that doesn't commit.
// interface in terms of noms because it will get called during sync, where we already have noms data.
func (db *DB) execImpl(function string, args types.List) error {
	oldBundle := db.head.Bundle(db.noms)
	newBundle := oldBundle
	oldData := db.head.Data(db.noms)
	ed := editor{db.noms, oldData.Edit()}

	commit := Commit{}
	commit.Parents = []types.Ref{types.NewRef(db.head.Original)}
	commit.Meta.Date = datetime.Now()
	commit.Meta.Tx.Origin = db.origin
	commit.Meta.Tx.Name = function
	commit.Meta.Tx.Args = args

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
		commit.Meta.Tx.Code = types.NewRef(oldBundle)
		err := exec.Run(ed, oldBundle.Reader(), function, args)
		if err != nil {
			return err
		}
	}

	newData := ed.Finalize()
	if newData.Equals(oldData) && newBundle.Equals(oldBundle) {
		return nil
	}

	commit.Value.Data = db.noms.WriteValue(newData)
	commit.Value.Code = db.noms.WriteValue(newBundle)
	commit.Original = marshal.MustMarshal(db.noms, commit).(types.Struct)

	nomsCommit := db.noms.WriteValue(commit.Original)

	// FastForward not strictly needed here because we should have already ensured that we were
	// fast-forwarding outside of Noms, but it's a nice sanity check.
	_, err := db.noms.FastForward(db.noms.GetDataset(local_dataset), nomsCommit)
	if err != nil {
		return err
	}
	db.head = commit
	return nil
}
