package kv

import (
	"bytes"
	"fmt"

	"roci.dev/diff-server/util/chk"

	"github.com/attic-labs/noms/go/types"
	nomsjson "github.com/attic-labs/noms/go/util/json"
)

// Map is a map from string key to json bytes. Map is
// NOT threadsafe.
type Map struct {
	noms types.ValueReadWriter
	nm   types.Map
	c    Checksum
}

// NewMap returns a new, empty Map.
func NewMap(noms types.ValueReadWriter) *Map {
	return &Map{noms, types.NewMap(noms), Checksum{0}}
}

// NewMapFromNoms returns a new Map from an existing noms map. This
// is mainly useful in testing, so far. Creates a full copy by iterating
// the noms map, so be careful.
func NewMapFromNoms(noms types.ValueReadWriter, nm types.Map) *Map {
	// We dont want to just return a Map with the embedded noms map because
	// that misses applying any logic in Map.Set eg canonicalization.
	m := NewMap(noms)
	me := m.Edit()
	for mi := nm.Iterator(); mi.Valid(); mi.Next() {
		k := string(mi.Key().(types.String))
		v, err := bytesFromNomsValue(mi.Value())
		if err != nil {
			chk.Fail("Failed to serialize value to json.")
		}

		me.Set(k, v)
	}
	return me.Map()
}

// Get returns the json bytes for the given key, which must exist.
// It is an error to Get a key that does not exist.
func (m Map) Get(key string) ([]byte, error) {
	value, ok := m.nm.MaybeGet(types.String(key))
	if !ok {
		return []byte{}, fmt.Errorf("no such key %s", key)
	}

	return bytesFromNomsValue(value)
}

func bytesFromNomsValue(value types.Valuable) ([]byte, error) {
	// Here we could check value.Kind() if we wanted.
	var b bytes.Buffer
	if err := nomsjson.ToJSON(value.Value(), &b, nomsjson.ToOptions{
		Lists: true,
		Maps:  true,
	}); err != nil {
		return []byte{}, err
	}

	// Canonicalize the output by round tripping through canonical json.
	// This is inefficient; noms should have the option to canonicalize.
	// This is also a ripe territory for bugs and we should have better
	// canonicalization testing.
	// TODO canonicalize using 	cjson "github.com/gibson042/canonicaljson-go"
	// TODO be sure to disallow nil values

	return b.Bytes(), nil
}

// Checksum is the Cheksum over the Map of k/vs.
func (m Map) Checksum() Checksum {
	return m.c
}

// Edit returns a MapEditor allowing mutation of the Map. The original
// Map is not affected.
func (m Map) Edit() *MapEditor {
	return &MapEditor{m.noms, m.nm.Edit(), m.c}
}

// String returns the string value of the Map.
func (m Map) String() string {
	return fmt.Sprintf("Checksum: %s, noms Map: %s\n", m.Checksum(), types.EncodedValue(m.nm))
}

// MapEditor allows mutation of a Map.
type MapEditor struct {
	noms types.ValueReadWriter
	nme  *types.MapEditor
	c    Checksum
}

// Get returns the value for a given key or an error if that key
// doesn't exist.
func (me MapEditor) Get(key string) ([]byte, error) {
	nk := types.String(key)
	if !me.nme.Has(nk) {
		return nil, fmt.Errorf("no such key %s", key)
	}

	value := me.nme.Get(nk)
	return bytesFromNomsValue(value)
}

// Set sets the value for a given key.
func (me *MapEditor) Set(key string, value []byte) error {
	// TODO use canonical json here.
	nv, err := nomsjson.FromJSON(bytes.NewReader(value), me.noms, nomsjson.FromOptions{})
	if err != nil {
		return err
	}

	nk := types.String(key)
	if me.nme.Has(nk) {
		// Have to do this in order to properly update checksum.
		me.Remove(key)
	}
	me.c.Add(key, value)
	me.nme.Set(nk, nv)
	return nil
}

// Remove removes a key from the Map. It is an error to remove a key that
// does not exist.
func (me *MapEditor) Remove(key string) error {
	nk := types.String(key)
	if !me.nme.Has(nk) {
		return fmt.Errorf("no such key %s", key)
	}

	// Need the old value to update the checksum.
	oldValue, err := me.Get(key)
	if err != nil {
		return err
	}
	me.c.Remove(key, oldValue)
	me.nme.Remove(nk)

	return nil
}

// Map converts back into a Map.
func (me *MapEditor) Map() *Map {
	return &Map{me.noms, me.nme.Map(), me.c}
}
