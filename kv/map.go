package kv

import (
	"bytes"
	"errors"
	"fmt"

	"roci.dev/diff-server/util/chk"

	"github.com/attic-labs/noms/go/types"
	cjson "github.com/gibson042/canonicaljson-go"
	nomsjson "roci.dev/diff-server/util/noms/json"
)

// Map is a map from string key to json bytes. Map is
// NOT threadsafe.
type Map struct {
	noms types.ValueReadWriter
	nm   types.Map
	sum  Checksum
}

// NewMap returns a new Map.
func NewMap(noms types.ValueReadWriter, kv ...string) Map {
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
		v, err := bytesFromNomsValue(mi.Value())
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
func (m Map) Has(key string) bool {
	return m.nm.Has(types.String(key))
}

// Get returns the canonical json bytes for the given key.
func (m Map) Get(key string) ([]byte, error) {
	value, ok := m.nm.MaybeGet(types.String(key))
	if !ok {
		return nil, nil
	}

	return bytesFromNomsValue(value)
}

// Empty returns true if the Map is empty.
func (m Map) Empty() bool {
	return m.nm.Empty()
}

func bytesFromNomsValue(value types.Valuable) ([]byte, error) {
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
func (me MapEditor) Has(key string) bool {
	return me.nme.Has(types.String(key))
}

// Get returns the canonical json bytes for the given key.
func (me MapEditor) Get(key string) ([]byte, error) {
	nk := types.String(key)
	if !me.nme.Has(nk) {
		return nil, nil
	}

	value := me.nme.Get(nk)
	return bytesFromNomsValue(value)
}

// Set sets the value for a given key. Set canonicalizes value.
func (me *MapEditor) Set(key string, value []byte) error {
	if key == "" {
		return errors.New("key must be non-empty")
	}

	// Round trip to canonicalize. Note in the following lines we are going:
	//     json -> go -> canonical json -> noms
	// It is tempting to try to save a step by going:
	//     json -> noms -> canonical json
	// and use the intermediate noms value. Unfortunately, we can't because
	// there is no guarantee that it would be the same as the noms value parsed
	// from the canonical json. For example, it might have unnormalized strings.
	var v interface{}
	if err := cjson.Unmarshal(value, &v); err != nil {
		return fmt.Errorf("couldnt parse value '%s' as json: %w", string(value), err)
	}
	canonicalizedValue, err := cjson.Marshal(v)
	if err != nil {
		return err
	}
	nv, err := nomsjson.FromJSON(bytes.NewReader(canonicalizedValue), me.noms)
	if err != nil {
		return err
	}

	nk := types.String(key)
	if me.nme.Has(nk) {
		// Have to do this in order to properly update checksum.
		if err := me.Remove(key); err != nil {
			return err
		}
	}

	me.nme.Set(nk, nv)
	me.sum.Add(key, canonicalizedValue)
	return nil
}

// Remove removes a key from the Map.
func (me *MapEditor) Remove(key string) error {
	nk := types.String(key)
	if !me.nme.Has(nk) {
		return nil
	}

	// Need the old value to update the checksum.
	oldValue, err := me.Get(key)
	if err != nil {
		return err
	}
	me.sum.Remove(key, oldValue)
	me.nme.Remove(nk)

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
