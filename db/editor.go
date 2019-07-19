package db

import (
	"github.com/attic-labs/noms/go/types"
)

type editor struct {
	noms types.ValueReadWriter
	data *types.MapEditor
}

func (ed editor) Noms() types.ValueReadWriter {
	return ed.noms
}

func (ed editor) Has(id string) (bool, error) {
	return ed.data.Has(types.String(id)), nil
}

func (ed editor) Get(id string) (types.Value, error) {
	return ed.data.Get(types.String(id)).Value(), nil
}

// This interface has to be in terms of values because sync is going to call it with values.
func (ed editor) Put(id string, v types.Value) error {
	ed.data.Set(types.String(id), v)
	return nil
}

func (ed editor) Del(id string) (bool, error) {
	if !ed.data.Has(types.String(id)) {
		return false, nil
	}
	ed.data.Remove(types.String(id))
	return true, nil
}

func (ed editor) Finalize() types.Map {
	return ed.data.Map()
}
