package kv_test

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/noms/memstore"
)

func TestNewMap(t *testing.T) {
	assert := assert.New(t)

	noms := memstore.New()

	// Ensure checksum matches if constructed vs built.
	nm := types.NewMap(noms, types.String("key"), types.String("1"))
	m := kv.WrapMapAndComputeChecksum(noms, nm)
	expectedm := kv.WrapMapAndComputeChecksum(noms, types.NewMap(noms))
	e := expectedm.Edit()
	assert.NoError(e.Set("key", []byte(" \"1\" "))) // note spaces intentional to ensure canonicalizes
	expectedm = e.Build()
	assert.True(expectedm.Sum.Equal(m.Sum), "got checksum %v, wanted %v", m.DebugString(), expectedm.DebugString())
}

func TestCanonicalizes(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	k, v := "k", []byte("  {  \"z\"   : 1, \n \"a\":    2  } \r")
	expectedv := []byte("{\"a\":2,\"z\":1}\n")

	// Does it appear to canonicalize?
	m := kv.NewMap(noms)
	me := m.Edit()
	assert.NoError(me.Set(k, v))
	assertGetEqual(assert, me, k, expectedv)
	m = me.Build()
	assertGetEqual(assert, m, k, expectedv)

	// Does canonicalized map match one where we set an already canonicalized value?
	// This is an extremely important test!
	m2 := kv.NewMap(noms)
	m2e := m2.Edit()
	assert.NoError(m2e.Set(k, expectedv))
	assertGetEqual(assert, m2e, k, expectedv)
	m2 = m2e.Build()
	assert.True(m2.Sum.Equal(m.Sum))
}

type getter interface {
	Get(string) ([]byte, error)
}

func assertGetEqual(assert *assert.Assertions, m getter, key string, expected []byte) {
	got, err := m.Get(key)
	assert.NoError(err)
	assert.Equal(expected, got)
}

func assertGetError(assert *assert.Assertions, m getter, key string) {
	_, err := m.Get(key)
	assert.Error(err, "no such key")
}

func TestMapGetSetRemove(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	k1 := "k1"
	v1, v2 := []byte("\"1\"\n"), []byte("\"2\"\n")

	em := kv.NewMap(noms)
	assertGetEqual(assert, em, k1, nil)

	m1 := kv.NewMap(noms)
	m1e := m1.Edit()
	assert.NoError(m1e.Set(k1, v1))
	assertGetEqual(assert, m1e, k1, v1)
	m1 = m1e.Build()
	assert.False(em.Sum.Equal(m1.Sum))
	assertGetEqual(assert, m1, k1, v1)
	m1e = m1.Edit()
	m1e.Set(k1, v2)
	assertGetEqual(assert, m1e, k1, v2)
	assertGetEqual(assert, m1, k1, v1)
	m2 := m1e.Build()
	assertGetEqual(assert, m2, k1, v2)
	assert.False(m2.Sum.Equal(m1.Sum))

	m2e := m2.Edit()
	m2e.Remove(k1)
	assertGetEqual(assert, m2e, k1, nil)
	assert.NoError(m2e.Remove(k1))
	m2got := m2e.Build()
	assertGetEqual(assert, m2got, k1, nil)
	assert.False(m2got.Sum.Equal(m2.Sum))
	assert.True(m2got.Sum.Equal(em.Sum), "got=%s, want=%s", m2got.DebugString(), em.DebugString())

	// Test that if we do two edit operations both stick.
	k2 := "k2"
	m1 = kv.NewMap(noms)
	m1e = m1.Edit()
	assert.NoError(m1e.Set(k1, v1))
	assert.NoError(m1e.Set(k2, v2))
	assertGetEqual(assert, m1e, k1, v1)
	assertGetEqual(assert, m1e, k2, v2)
	m1 = m1e.Build()
	assertGetEqual(assert, m1, k1, v1)
	assertGetEqual(assert, m1, k2, v2)
}
func TestNull(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()
	m1 := kv.NewMap(noms)
	m1e := m1.Edit()
	err := m1e.Set("foo", []byte("null"))
	m1 = m1e.Build()
	assert.NoError(err)
	act, err := m1.Get("foo")
	assert.NoError(err)
	assert.Equal([]byte("null\n"), act)
}

func TestEmptyKey(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()
	me := kv.NewMap(noms).Edit()
	assert.Error(me.Set("", []byte("true")), "key must be non-empty")
}

func TestEmpty(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	m := kv.NewMap(noms)
	assert.True(m.Empty())
	me := kv.NewMap(noms).Edit()
	assert.NoError(me.Set("foo", []byte("true")))
	m = me.Build()
	assert.False(m.Empty())
}
