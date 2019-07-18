package db

import (
	"errors"
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/json"
)

type editor struct {
	vrw  types.ValueReadWriter
	data *types.MapEditor
	code types.Blob
}

func (ed editor) Has(id string) (bool, error) {
	return ed.data.Has(types.String(id)), nil
}

func (ed editor) Get(id string, w io.Writer) (bool, error) {
	vv := ed.data.Get(types.String(id))
	if vv == nil {
		return false, nil
	}
	return streamGet(id, vv.Value(), w)
}

func (ed editor) Put(id string, r io.Reader) error {
	v, err := json.FromJSON(r, ed.vrw, json.FromOptions{})
	if err != nil {
		return fmt.Errorf("Invalid JSON: %s", err.Error())
	}
	if v == nil {
		return errors.New("Cannot write null")
	}
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

func (ed editor) PutCode(b types.Blob) {
	ed.code = b
}

func (ed editor) Finalize() (types.Map, types.Blob) {
	return ed.data.Map(), ed.code
}
