package kv_test

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/noms/memstore"
)

func b(s string) []byte {
	return []byte(s)
}

func TestComputeChecksum(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	// Ensure it matches when built.
	me := kv.NewMap(noms).Edit()
	assert.NoError(me.Set("foo", b("true")))
	assert.NoError(me.Set("bar", b("true")))
	assert.NoError(me.Remove("foo"))
	m := me.Build()
	assert.Equal(m.Checksum(), kv.ComputeChecksum(m.NomsMap()).String())

	// Ensure it matches a noms map separately constructed.
	nm := types.NewMap(noms, types.String("bar"), types.Bool(true))
	assert.Equal(m.Checksum(), kv.ComputeChecksum(nm).String())
}

func TestCanonicalizes(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	k, v := "k", []byte("  {  \"z\"   : 1, \n \"a\":    2  } \r")
	expectedv := []byte("{\"a\":2,\"z\":1}")

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
	assert.Equal(m.Checksum(), m2.Checksum())
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

func TestHas(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	k := "key"
	v := []byte("true")

	m := kv.NewMap(noms)
	assert.False(m.Has(k))
	me := m.Edit()
	assert.False(me.Has(k))
	assert.NoError(me.Set(k, v))
	assert.True(me.Has(k))
	assert.False(m.Has(k))
	assert.NoError(me.Remove(k))
	assert.False(m.Has(k))
	assert.NoError(me.Set(k, v))
	m = me.Build()
	assert.True(m.Has(k))
}

func TestMapGetSetRemove(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	k1 := "k1"
	v1, v2 := []byte("\"1\""), []byte("\"2\"")

	em := kv.NewMap(noms)
	assertGetEqual(assert, em, k1, nil)

	m1 := kv.NewMap(noms)
	m1e := m1.Edit()
	assert.NoError(m1e.Set(k1, v1))
	assertGetEqual(assert, m1e, k1, v1)
	m1 = m1e.Build()
	assert.NotEqual(em.Checksum(), m1.Checksum())
	assertGetEqual(assert, m1, k1, v1)
	m1e = m1.Edit()
	m1e.Set(k1, v2)
	assertGetEqual(assert, m1e, k1, v2)
	assertGetEqual(assert, m1, k1, v1)
	m2 := m1e.Build()
	assertGetEqual(assert, m2, k1, v2)
	assert.NotEqual(m2.Checksum(), m1.Checksum())

	m2e := m2.Edit()
	m2e.Remove(k1)
	assertGetEqual(assert, m2e, k1, nil)
	assert.NoError(m2e.Remove(k1))
	m2got := m2e.Build()
	assertGetEqual(assert, m2got, k1, nil)
	assert.NotEqual(m2.Checksum(), m2got.Checksum())
	assert.Equal(em.Checksum(), m2got.Checksum(), "got=%s, want=%s", m2got.DebugString(), em.DebugString())

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
	assert.Equal([]byte("null"), act)
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
