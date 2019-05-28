package db

import (
	"errors"
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/attic-labs/noms/go/util/json"
)

var (
	schema = nomdl.MustParseType(`Struct {
	code?: Ref<Blob>,
	data?: Ref<Map<String, Value>>,
}`)

	errCodeNotFound = errors.New("not found")
)

// Not thread-safe
// TODO: need to think carefully about concurrency here
type DB struct {
	db       datas.Database
	ds       datas.Dataset
	data     *types.MapEditor
	code     types.Blob
	origData types.Map
	origCode types.Blob
}

func Load(sp spec.Spec) (*DB, error) {
	db := sp.GetDatabase()
	ds := db.GetDataset("local")
	hv, ok := ds.MaybeHeadValue()
	data := types.NewMap(db)
	code := types.NewEmptyBlob(db)
	if !ok {
		return &DB{db, ds, data.Edit(), code, data, code}, nil
	}
	hvt := types.TypeOf(hv)
	if !types.IsSubtype(schema, hvt) {
		return &DB{}, fmt.Errorf("Dataset '%s::local' exists and has non-Replicant data of type: %s", sp.String(), hvt.Describe())
	}
	h := hv.(types.Struct)
	data = h.Get("data").(types.Ref).TargetValue(db).(types.Map)
	code = h.Get("code").(types.Ref).TargetValue(db).(types.Blob)
	return &DB{db, ds, data.Edit(), code, data, code}, nil
}

func (db DB) Noms() types.ValueReadWriter {
	return db.db
}

func (db *DB) Put(id string, r io.Reader) error {
	v, err := json.FromJSON(r, db.db, json.FromOptions{})
	if err != nil {
		return fmt.Errorf("Invalid JSON: %s", err.Error())
	}
	if v == nil {
		return errors.New("Cannot write null")
	}
	db.data.Set(types.String(id), v)
	return nil
}

func (db *DB) Has(id string) (bool, error) {
	return db.data.Has(types.String(id)), nil
}

func (db *DB) Get(id string, w io.Writer) (bool, error) {
	vv := db.data.Get(types.String(id))
	if vv == nil {
		return false, nil
	}
	v := vv.Value()
	err := json.ToJSON(v, w, json.ToOptions{
		Lists:  true,
		Maps:   true,
		Indent: "",
	})
	if err != nil {
		return false, fmt.Errorf("Key '%s' has non-Replicant data of type: %s", id, types.TypeOf(v).Describe())
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) Del(id string) (bool, error) {
	if !db.data.Has(types.String(id)) {
		return false, nil
	}
	db.data.Remove(types.String(id))
	return true, nil
}

func (db *DB) PutCode(b types.Blob) error {
	db.code = b
	return nil
}

func (db *DB) GetCode() (io.Reader, error) {
	if db.code == (types.Blob{}) {
		return nil, errors.New("no code bundle is registered")
	}
	return db.code.Reader(), nil
}

func (db *DB) Commit(origin, fn string, args types.List, date datetime.DateTime) error {
	data := db.data.Map()
	if data.Equals(db.origData) && db.code.Equals(db.origCode) {
		return nil
	}

	h := types.NewStruct("", types.StructData{
		"data": db.db.WriteValue(db.data.Map()),
		"code": db.db.WriteValue(db.code),
	})
	d, err := date.MarshalNoms(db.Noms())
	if err != nil {
		return err
	}
	opts := datas.CommitOptions{
		Parents: types.Set{},
		Meta: types.NewStruct("", types.StructData{
			"date": d,
			"tx": types.NewStruct("", types.StructData{
				"origin": types.String(origin),
				"name":   types.String(fn),
				"args":   args,
			}),
		}),
	}

	ds, err := db.db.Commit(db.ds, h, opts)
	if err != nil {
		return err
	}
	db.ds = ds
	return nil
}
