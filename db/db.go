package db

import (
	"errors"
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/json"
)

var (
	schema = nomdl.MustParseType(`Struct {
	code: Ref<Set<Blob>>,
	data: Ref<Map<String, Value>>,
}`)

	errCodeNotFound = errors.New("not found")
)

// Not thread-safe
// TODO: need to think carefully about concurrency here
type DB struct {
	db   datas.Database
	ds   datas.Dataset
	data *types.MapEditor
	code *types.SetEditor
}

func Load(sp spec.Spec) (DB, error) {
	db := sp.GetDatabase()
	ds := db.GetDataset("local")
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return DB{db, ds, types.NewMap(db).Edit(), types.NewSet(db).Edit()}, nil
	}
	hvt := types.TypeOf(hv)
	if !types.IsSubtype(schema, hvt) {
		return DB{}, fmt.Errorf("Dataset '%s::local' exists and has non-Replicant data of type: %s", sp.String(), hvt.Describe())
	}
	h := hv.(types.Struct)
	data := h.Get("data").(types.Ref).TargetValue(db).(types.Map).Edit()
	code := h.Get("code").(types.Ref).TargetValue(db).(types.Set).Edit()
	return DB{db, ds, data, code}, nil
}

func (db DB) Put(id string, r io.Reader) error {
	v, err := json.FromJSON(r, db.db, json.FromOptions{})
	if err != nil {
		return err
	}
	db.data.Set(types.String(id), v)
	return nil
}

func (db DB) Has(id string) (bool, error) {
	return db.data.Has(types.String(id)), nil
}

func (db DB) Get(id string, w io.Writer) (bool, error) {
	vv := db.data.Get(types.String(id))
	if vv == nil {
		return false, nil
	}
	v := vv.Value()
	err := json.ToJSON(v, w, json.ToOptions{
		Lists:  true,
		Maps:   true,
		Indent: "  ",
	})
	if err != nil {
		return false, fmt.Errorf("Key '%s' has non-Replicant data of type: %s", types.TypeOf(v).Describe())
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db DB) Del(id string) (bool, error) {
	if !db.data.Has(types.String(id)) {
		return false, nil
	}
	db.data.Remove(types.String(id))
	return true, nil
}

func (db DB) PutCode(r io.Reader) (hash.Hash, error) {
	// TODO: Do we want to validate that it compiles or whatever???
	b := types.NewBlob(db.db, r)
	db.code.Insert(b)
	return b.Hash(), nil
}

func (db DB) HasCode(hash hash.Hash) (bool, error) {
	_, err := db.GetCode(hash)
	if err == nil {
		return true, nil
	} else if err == errCodeNotFound {
		return false, nil
	} else {
		return false, err
	}
}

func (db DB) GetCode(hash hash.Hash) (types.Blob, error) {
	it := db.code.Set().Iterator()
	for v := it.Next(); v != nil; v = it.Next() {
		if v.Hash() == hash {
			// Downcast here is known to be safe because we checked schema in Load().
			return v.(types.Blob), nil
		}
	}

	return types.Blob{}, errCodeNotFound
}

func (db DB) DelCode(hash hash.Hash) (bool, error) {
	v, err := db.GetCode(hash)
	if err == errCodeNotFound {
		return false, nil
	} else {
		return false, err
	}
	db.code.Remove(v)
	return true, nil
}

func (db DB) ListCode() (chan types.Blob, error) {
	r := make(chan types.Blob)
	it := db.code.Set().Iterator()
	go func() {
		for v := it.Next(); v != nil; v = it.Next() {
			r <- v.(types.Blob)
		}
		close(r)
	}()
	return r, nil
}

func (db DB) Commit() error {
	h := types.NewStruct("", types.StructData{
		"data": db.db.WriteValue(db.data.Map()),
		"code": db.db.WriteValue(db.code.Set()),
	})
	ds, err := db.db.CommitValue(db.ds, h)
	if err != nil {
		return err
	}
	db.ds = ds
	return nil
}

func parseHash(h string) (hash.Hash, error) {
	r, ok := hash.MaybeParse(h)
	if !ok {
		return hash.Hash{}, errors.New("Invalid hash string")
	}
	return r, nil
}
