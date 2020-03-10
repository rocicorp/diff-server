package kv

import (
	"bytes"
	"fmt"

	"roci.dev/diff-server/util/chk"

	"github.com/attic-labs/noms/go/types"
	nomsjson "roci.dev/diff-server/util/noms/json"
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
	return me.Build()
}

// NomsMap returns the underlying noms map.
func (m Map) NomsMap() types.Map {
	return m.nm
}

// Get returns the json bytes for the given key, which must exist.
func (m Map) Get(key string) ([]byte, error) {
	value, ok := m.nm.MaybeGet(types.String(key))
	if !ok {
		return nil, nil
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

	// nomsjson uses Encoder, which assumes it is encoding a stream
	// of values and therefore appends a newline. Values in our store
	// can only be a single json value. Here we trim the newline.
	bb := b.Bytes()
	if len(bb) == 0 || bb[len(bb)-1] != '\n' {
		chk.Fail("unexpected json encoding: %s", bb)
	}
	bb = bb[:len(bb)-1]
	return bb, nil
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
func (me MapEditor) Get(key string) ([]byte, error) {
	nk := types.String(key)
	if !me.nme.Has(nk) {
		return nil, nil
	}

	value := me.nme.Get(nk)
	return bytesFromNomsValue(value)
}

// Set sets the value for a given key.
func (me *MapEditor) Set(key string, value []byte) error {
	nv, err := nomsjson.FromJSON(bytes.NewReader(value), me.noms, nomsjson.FromOptions{})
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

	// The value passed in might not be canonicalized. We have to round trip
	// to get the canonicalized version which is what we want to checksum and store.
	canonicalBytes, err := bytesFromNomsValue(nv)
	if err != nil {
		return err
	}
	// Note: here we are using the noms value unmarshalled from the *un*canonicalized json.
	// Might be safer to unmarshal here from the canonical json and use that value instead?
	me.nme.Set(nk, nv)
	me.c.Add(key, canonicalBytes)
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
	me.c.Remove(key, oldValue)
	me.nme.Remove(nk)

	return nil
}

// Build converts back into a Map.
func (me *MapEditor) Build() *Map {
	return &Map{me.noms, me.nme.Map(), me.c}
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
