package db

import (
	"bufio"
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

func (db DB) Get(id string, w io.Writer) error {
	vv := db.data.Get(types.String(id))
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

func (db DB) PutFunc(r io.Reader, w io.Writer) error {
	// TODO: Do we want to validate that it compiles or whatever???
	b := types.NewBlob(db.db, r)
	db.code.Insert(b)
	bw := bufio.NewWriter(w)
	bw.WriteString(b.Hash().String())
	bw.WriteByte('\n')
	bw.Flush()
	return nil
}

func (db DB) GetFunc(hash string) (types.Blob, error) {
	h, err := parseHash(hash)
	if err != nil {
		return types.Blob{}, err
	}

	it := db.code.Set().Iterator()
	for v := it.Next(); v != nil; v = it.Next() {
		if v.Hash() == h {
			// Downcast here is known to be safe because we checked schema in Load().
			return v.(types.Blob), err
		}
	}

	return types.Blob{}, errors.New("func not found")
}

func (db DB) DelFunc(hash string) error {
	v, err := db.GetFunc(hash)
	if err != nil {
		return err
	}
	db.code.Remove(v)
	return nil
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
