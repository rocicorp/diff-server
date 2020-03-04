package kv_test

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/noms/memstore"
)

// TODO unskip when canonicalization works.

func TestNewMap(t *testing.T) {
	t.Skip()

	assert := assert.New(t)

	noms := memstore.New()

	// Ensure checksum matches if constructed vs built.
	nm := types.NewMap(noms, types.String("key"), types.String("1"))
	m := kv.NewMapFromNoms(noms, nm)
	expectedm := kv.NewMapFromNoms(noms, types.NewMap(noms))
	e := expectedm.Edit()
	assert.NoError(e.Set("key", []byte("\"1\"")))
	expectedm = e.Build()
	assert.True(expectedm.Checksum().Equal(m.Checksum()), "got checksum %v, wanted %v", m.Checksum(), expectedm.Checksum())
}

type getter interface {
	Get(string) ([]byte, error)
}

func assertGetEqual(assert *assert.Assertions, m getter, key string, expected []byte) {
	got, err := m.Get(key)
	assert.NoError(err)
	assert.Equal(got, expected)
}

func assertGetError(assert *assert.Assertions, m getter, key string) {
	_, err := m.Get(key)
	assert.Error(err, "no such key")
}

func TestMapGetSetRemove(t *testing.T) {
	t.Skip()

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
	assert.False(em.Checksum().Equal(m1.Checksum()))
	assertGetEqual(assert, m1, k1, v1)
	m1e = m1.Edit()
	m1e.Set(k1, v2)
	assertGetEqual(assert, m1e, k1, v2)
	assertGetEqual(assert, m1, k1, v1)
	m2 := m1e.Build()
	assertGetEqual(assert, m2, k1, v2)
	assert.False(m2.Checksum().Equal(m1.Checksum()))

	m2e := m2.Edit()
	m2e.Remove(k1)
	assertGetEqual(assert, m2e, k1, nil)
	assert.NoError(m2e.Remove(k1))
	m2got := m2e.Build()
	assertGetEqual(assert, m2got, k1, nil)
	assert.False(m2got.Checksum().Equal(m2.Checksum()))
	assert.True(m2got.Checksum().Equal(em.Checksum()), "got=%s, want=%s", m2got.DebugString(), em.DebugString())

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
