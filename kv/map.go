package kv

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

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
	Sum  Checksum
}

// NewMap returns a new, empty Map. This constructor will panic if there is
// an error setting any of the values, so don't use it with user data.
func NewMap(noms types.ValueReadWriter, kv ...string) Map {
	me := Map{noms, types.NewMap(noms), Checksum{0}}.Edit()
	for i := 0; i < len(kv); i += 2 {
		err := me.Set(kv[i], []byte(kv[i+1]))
		chk.NoError(err)
	}
	return me.Build()
}

// WrapMap wraps a noms Map with additional logic replicache needs.
func WrapMap(noms types.ValueReadWriter, nm types.Map, c Checksum) Map {
	return Map{noms, nm, c}
}

// NewMapFromPile returns a new map from a pile (which was presumably decoded from json).
func NewMapFromPile(noms types.ValueReadWriter, pile map[string]interface{}) (Map, error) {
	me := NewMap(noms).Edit()
	for k, v := range pile {
		canonicalizedValue, err := cjson.Marshal(v)
		if err != nil {
			return Map{}, fmt.Errorf("failed to convert value for key %s to json", k)
		}
		if err := me.SetCanonicalized(k, canonicalizedValue); err != nil {
			return Map{}, err
		}
	}
	return me.Build(), nil
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
		// noms adds a newline when encoding, so strip it.
		v = []byte(strings.TrimRight(string(v), "\n"))
		c.Add(k, v)
	}
	return c
}

// NomsMap returns the underlying noms map.
func (m Map) NomsMap() types.Map {
	return m.nm
}

// Get returns the json bytes for the given key, which must exist.
// Note the json bytes gotten include a trailing newline.
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
	// Here we could check value.Kind() if we wanted.
	var b bytes.Buffer
	if err := nomsjson.ToJSON(value.Value(), &b); err != nil {
		return []byte{}, err
	}
	return b.Bytes(), nil
}

// Checksum is the checksum of the Map.
func (m Map) Checksum() string {
	return m.Sum.String()
}

// NomsChecksum returns the checksum as a types.String.
func (m Map) NomsChecksum() types.String {
	return types.String(m.Checksum())
}

// Edit returns a MapEditor allowing mutation of the Map. The original
// Map is not affected.
func (m Map) Edit() *MapEditor {
	return &MapEditor{m.noms, m.nm.Edit(), m.Sum}
}

// DebugString returns a nice string value of the Map, including the full underlying noms map.
func (m Map) DebugString() string {
	return fmt.Sprintf("Checksum: %s, noms Map: %v\n", m.Checksum(), types.EncodedValue(m.nm))
}

// MapEditor allows mutation of a Map.
type MapEditor struct {
	noms types.ValueReadWriter
	nme  *types.MapEditor
	c    Checksum
}

// Get returns the value for a given key or an error if that key
// doesn't exist.
// Note the json bytes gotten include a trailing newline.
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
	var v interface{}
	if err := cjson.Unmarshal(value, &v); err != nil {
		return err
	}
	canonicalizedValue, err := cjson.Marshal(v)
	if err != nil {
		return err
	}
	return me.SetCanonicalized(key, canonicalizedValue)
}

// SetCanonicalized sets the value for a given key. It assumes value is canonicalized.
func (me *MapEditor) SetCanonicalized(key string, value []byte) error {
	if key == "" {
		return errors.New("key must be non-empty")
	}

	nv, err := nomsjson.FromJSON(bytes.NewReader(value), me.noms)
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
	me.c.Add(key, value)
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
	// Strip the newline Get includes.
	oldValue = []byte(strings.TrimRight(string(oldValue), "\n"))
	me.c.Remove(key, oldValue)
	me.nme.Remove(nk)

	return nil
}

// Build converts back into a Map.
func (me *MapEditor) Build() Map {
	return Map{me.noms, me.nme.Map(), me.c}
}

// Checksum is the Cheksum over the Map of k/vs.
func (me MapEditor) Checksum() Checksum {
	return me.c
}

// DebugString returns a nice string value of the MapEditor, including the full underlying noms map.
func (me MapEditor) DebugString() string {
	m := me.Build()
	return fmt.Sprintf("Checksum: %s, noms Map: %v\n", m.Checksum(), types.EncodedValue(m.nm))
}
