package kv

import (
	"bytes"
	"fmt"

	"roci.dev/diff-server/util/chk"
	nomsjson "roci.dev/diff-server/util/noms/json"

	"github.com/attic-labs/noms/go/types"
)

// Map embeds types.Map, adding a few bits of logic like checksumming.
// Map is NOT threadsafe.
type Map struct {
	noms types.ValueReadWriter
	types.Map
	sum Checksum
}

// NewMap returns a new Map.
func NewMap(noms types.ValueReadWriter) Map {
	return Map{noms, types.NewMap(noms), Checksum{0}}
}

// FromNoms creates a map from an existing Noms Map and Checksum.
func FromNoms(noms types.ValueReadWriter, nm types.Map, c Checksum) Map {
	return Map{noms, nm, c}
}

// ComputeChecksum iterates a noms map and computes its checksum. The noms map is
// assumed to be canonicalized.
func ComputeChecksum(nm types.Map) Checksum {
	c := Checksum{}
	for mi := nm.Iterator(); mi.Valid(); mi.Next() {
		k := string(mi.Key().(types.String))
		v, err := toJSON(mi.Value())
		if err != nil {
			chk.Fail("Failed to serialize value to json.")
		}
		c.Add(k, v)
	}
	return c
}

// NomsMap returns the underlying noms map.
func (m Map) NomsMap() types.Map {
	return m.Map
}

func toJSON(value types.Valuable) ([]byte, error) {
	var b bytes.Buffer
	if err := nomsjson.ToJSON(value.Value(), &b); err != nil {
		return []byte{}, err
	}
	return b.Bytes(), nil
}

// Checksum is the checksum of the Map.
func (m Map) Checksum() string {
	return m.sum.String()
}

// NomsChecksum returns the checksum as a types.String.
func (m Map) NomsChecksum() types.String {
	return types.String(m.Checksum())
}

// Edit returns a MapEditor allowing mutation of the Map. The original
// Map is not affected.
func (m Map) Edit() *MapEditor {
	return &MapEditor{m.noms, m.Map.Edit(), m.sum}
}

// DebugString returns a nice string value of the Map, including the full underlying noms map.
func (m Map) DebugString() string {
	return fmt.Sprintf("Checksum: %s, noms Map: %v\n", m.Checksum(), types.EncodedValue(m.NomsMap()))
}

// MapEditor embeds a types.MapEditor, enabling mutations.
type MapEditor struct {
	noms types.ValueReadWriter
	*types.MapEditor
	sum Checksum
}

// Get changes the signature of MapEditor's Get to match that of Map.
func (me *MapEditor) Get(key types.Value) types.Value {
	v := me.MapEditor.Get(key)
	if v == nil {
		return nil
	}
	return v.Value()
}

// Set sets the value for a given key. Set requires that the value has been
// be parsed from canonical json, otherwise we might parse two different
// values for the same canonical json.
func (me *MapEditor) Set(key types.String, value types.Value) error {
	if me.MapEditor.Has(key) {
		// Have to do this in order to properly update checksum.
		if err := me.Remove(key); err != nil {
			return err
		}
	}

	JSON, err := toJSON(value)
	if err != nil {
		return err
	}
	me.MapEditor.Set(key, value)
	me.sum.Add(string(key), JSON)
	return nil
}

// Remove removes a key from the Map.
func (me *MapEditor) Remove(key types.String) error {
	// Need the old value to update the checksum.
	// Note: Noms MapEditor.Get can return a value that has been removed
	// so here we check Has, which works correctly. Once
	// https://github.com/attic-labs/noms/pull/3872 is released this can
	// just Get directly.
	if me.MapEditor.Has(key) {
		oldValue := me.MapEditor.Get(key)
		oldValueJSON, err := toJSON(oldValue.Value())
		if err != nil {
			return err
		}
		me.sum.Remove(string(key), oldValueJSON)
	}

	me.MapEditor.Remove(key)
	return nil
}

// Build converts back into a Map.
func (me *MapEditor) Build() Map {
	return Map{me.noms, me.MapEditor.Map(), me.sum}
}

// Checksum is the Cheksum over the Map of k/vs.
func (me MapEditor) Checksum() Checksum {
	return me.sum
}

// DebugString returns a nice string value of the MapEditor, including the full underlying noms map.
func (me MapEditor) DebugString() string {
	m := me.Build()
	return fmt.Sprintf("Checksum: %s, noms Map: %v\n", m.Checksum(), types.EncodedValue(m.NomsMap()))
}
