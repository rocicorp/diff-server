package kv_test

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/kv"
	nomsjson "roci.dev/diff-server/util/noms/json"
	"roci.dev/diff-server/util/noms/memstore"
)

func s(s string) types.String {
	return types.String(s)
}

func TestComputeChecksum(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	// Ensure it matches when built.
	me := kv.NewMap(noms).Edit()
	assert.NoError(me.Set(s("foo"), types.Bool(true)))
	assert.NoError(me.Set(s("bar"), types.Bool(true)))
	assert.NoError(me.Remove(s("foo")))
	m := me.Build()
	assert.Equal(m.Checksum(), kv.ComputeChecksum(m.NomsMap()).String())

	// Ensure it matches a noms map separately constructed.
	nm := types.NewMap(noms, s("bar"), types.Bool(true))
	assert.Equal(m.Checksum(), kv.ComputeChecksum(nm).String())
}

type getter interface {
	Get(types.Value) types.Value
}

func assertGetEqual(assert *assert.Assertions, m getter, key types.String, expected types.Value) {
	got := m.Get(key)
	if expected == nil {
		assert.Nil(got)
	} else {
		assert.True(expected.Equals(got))
	}
}

func TestMapGetSetRemove(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	k1 := s("k1")
	v1, v2 := s("1"), s("2")

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
	assert.NoError(m2e.Remove(k1))
	// Uncomment when https://github.com/attic-labs/noms/pull/3872 is released.
	// assertGetEqual(assert, m2e, k1, nil)
	assert.NoError(m2e.Remove(k1))
	m2got := m2e.Build()
	assertGetEqual(assert, m2got, k1, nil)
	assert.NotEqual(m2.Checksum(), m2got.Checksum(), "got=%s, want=%s", m2got.DebugString(), m2.DebugString())
	assert.Equal(em.Checksum(), m2got.Checksum(), "got=%s, want=%s", m2got.DebugString(), em.DebugString())

	// Test that if we do two edit operations both stick.
	k2 := s("k2")
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
	err := m1e.Set(s("foo"), nomsjson.Null())
	m1 = m1e.Build()
	assert.NoError(err)
	act := m1.Get(s("foo"))
	assert.NotNil(act)
	assert.True(nomsjson.Null().Equals(act))
}

func TestEmptyKey(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()
	me := kv.NewMap(noms).Edit()
	assert.Error(me.Set(s(""), types.Bool(true)), "key must be non-empty")
}

func TestEmpty(t *testing.T) {
	assert := assert.New(t)
	noms := memstore.New()

	m := kv.NewMap(noms)
	assert.True(m.Empty())
	me := kv.NewMap(noms).Edit()
	assert.NoError(me.Set(s("foo"), types.Bool(true)))
	m = me.Build()
	assert.False(m.Empty())
}
