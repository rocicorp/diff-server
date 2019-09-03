package db

import (
	"github.com/aboodman/replicant/exec"
	"github.com/attic-labs/noms/go/types"
)

type editor struct {
	noms types.ValueReadWriter
	data *types.MapEditor

	// True if the instance has ever received a mutating call.
	// Note this is true even if the mutating call didn't succeed.
	receivedMutAttempt bool
}

func (ed editor) Noms() types.ValueReadWriter {
	return ed.noms
}

func (ed editor) Has(id string) (bool, error) {
	return ed.data.Has(types.String(id)), nil
}

func (ed editor) Get(id string) (types.Value, error) {
	vv := ed.data.Get(types.String(id))
	if vv == nil {
		return nil, nil
	}
	return vv.Value(), nil
}

func (ed editor) Scan(opts exec.ScanOptions) (r []exec.ScanItem, err error) {
	// Blech, this sucks. We need to build the map because Noms MapEditor doesn't support scans.
	// Implementing them is more effort than I have avaiable right now.
	m := ed.data.Map()
	r, err = scan(m, opts)
	ed.data = m.Edit()
	return
}

// This interface has to be in terms of values because sync is going to call it with values.
func (ed *editor) Put(id string, v types.Value) error {
	ed.receivedMutAttempt = true
	ed.data.Set(types.String(id), v)
	return nil
}

func (ed *editor) Del(id string) (bool, error) {
	// We are specifically tracking whether the user attempted to write, not whether any
	// changed happened (if we only wanted to know the latter we could just compare the
	// resulting end state).
	ed.receivedMutAttempt = true
	if !ed.data.Has(types.String(id)) {
		return false, nil
	}
	ed.data.Remove(types.String(id))
	return true, nil
}

func (ed editor) Finalize() types.Map {
	return ed.data.Map()
}
