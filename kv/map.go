package kv

import (
	"bytes"
	"errors"
	"fmt"

	"roci.dev/diff-server/util/chk"
	nomsjson "roci.dev/diff-server/util/noms/json"

	"github.com/attic-labs/noms/go/types"
)

// Map is a map from String key to Map representing JSON.
// Map is NOT threadsafe.
type Map struct {
	noms types.ValueReadWriter
	nm   types.Map
	sum  Checksum
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
	return m.nm
}

// Has returns true if there exists a value for the given key.
func (m Map) Has(key types.String) bool {
	return m.nm.Has(key)
}

// Get returns the Value for the given key and a bool indicating
// if it was gotten.
func (m Map) Get(key types.String) (types.Value, bool) {
	return m.nm.MaybeGet(key)
}

// Empty returns true if the Map is empty.
func (m Map) Empty() bool {
	return m.nm.Empty()
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
	return &MapEditor{m.noms, m.nm.Edit(), m.sum}
}

// DebugString returns a nice string value of the Map, including the full underlying noms map.
func (m Map) DebugString() string {
	return fmt.Sprintf("Checksum: %s, noms Map: %v\n", m.Checksum(), types.EncodedValue(m.nm))
}

// MapEditor allows mutation of a Map.
type MapEditor struct {
	noms types.ValueReadWriter
	nme  *types.MapEditor
	sum  Checksum
}

// Has returns true if there exists a value for the given key.
func (me MapEditor) Has(key types.String) bool {
	return me.nme.Has(key)
}

// Get returns the Value for the given key and a bool indicating if it was gotten.
func (me MapEditor) Get(key types.String) (types.Value, bool) {
	if !me.nme.Has(key) {
		return nil, false
	}

	return me.nme.Get(key).Value(), true
}

// Set sets the value for a given key. Set requires that the value has been
// be parsed from canonical json, otherwise we might parse two different
// values for the same canonical json.
func (me *MapEditor) Set(key types.String, value types.Value) error {
	if key == "" {
		return errors.New("key must be non-empty")
	}
	if me.nme.Has(key) {
		// Have to do this in order to properly update checksum.
		if err := me.Remove(key); err != nil {
			return err
		}
	}

	JSON, err := toJSON(value)
	if err != nil {
		return err
	}
	me.nme.Set(key, value)
	me.sum.Add(string(key), JSON)
	return nil
}

// Remove removes a key from the Map.
func (me *MapEditor) Remove(key types.String) error {
	// Need the old value to update the checksum.
	oldValue, got := me.Get(key)
	if !got {
		return nil
	}
	oldValueJSON, err := toJSON(oldValue)
	if err != nil {
		return err
	}
	me.sum.Remove(string(key), oldValueJSON)
	me.nme.Remove(key)

	return nil
}

// Build converts back into a Map.
func (me *MapEditor) Build() Map {
	return Map{me.noms, me.nme.Map(), me.sum}
}

// Checksum is the Cheksum over the Map of k/vs.
func (me MapEditor) Checksum() Checksum {
	return me.sum
}

// DebugString returns a nice string value of the MapEditor, including the full underlying noms map.
func (me MapEditor) DebugString() string {
	m := me.Build()
	return fmt.Sprintf("Checksum: %s, noms Map: %v\n", m.Checksum(), types.EncodedValue(m.nm))
}
