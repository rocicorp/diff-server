package db

import (
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/json"
)

var (
	objsMapType = nomdl.MustParseType(`Struct {data: Map<String, Value>}`)
)

// Not thread-safe
// TODO: need to think carefully about concurrency here
type DB struct {
	db   datas.Database
	ds   datas.Dataset
	h    types.Struct
	objs *types.MapEditor
}

func Load(sp spec.Spec) (DB, error) {
	db := sp.GetDatabase()
	ds := db.GetDataset("local")
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return DB{db, ds, types.NewStruct("", types.StructData{}), types.NewMap(db).Edit()}, nil
	}
	hvt := types.TypeOf(hv)
	// TODO: Check type of entire commit?
	if !types.IsSubtype(objsMapType, hvt) {
		return DB{}, fmt.Errorf("Dataset '%s::local' exists and has non-Replicant data of type: %s", sp.String(), hvt.Describe())
	}
	h := hv.(types.Struct)
	m := h.Get("data").(types.Map)
	return DB{db, ds, h, m.Edit()}, nil
}

func (db DB) Put(id string, r io.Reader) error {
	v, err := json.FromJSON(r, db.db, json.FromOptions{})
	if err != nil {
		return err
	}
	db.objs.Set(types.String(id), v)
	return nil
}

func (db DB) Get(id string, w io.Writer) error {
	vv := db.objs.Get(types.String(id))
	if vv == nil {
		return nil
	}
	v := vv.Value()
	err := json.ToJSON(v, w, json.ToOptions{
		Lists:  true,
		Maps:   true,
		Indent: "  ",
	})
	if err != nil {
		return fmt.Errorf("Key '%s' has non-Replicant data of type: %s", types.TypeOf(v).Describe())
	}
	return nil
}

func (db DB) Commit() error {
	h := db.h.Set("data", db.objs.Value())
	ds, err := db.db.CommitValue(db.ds, h)
	if err != nil {
		return err
	}
	db.ds = ds
	db.h = h
	return nil
}
