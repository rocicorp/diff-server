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
	expectedm = e.Map()
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
	//v1, v2 := []byte("\"1\""), []byte("\"2\"")
	v1, v2 := []byte("1"), []byte("2")

	em := kv.NewMap(noms)
	assertGetError(assert, em, k1)

	m1 := kv.NewMap(noms)
	m1e := m1.Edit()
	assert.NoError(m1e.Set(k1, v1))
	assertGetEqual(assert, m1e, k1, v1)
	m1 = m1e.Map()
	assert.False(em.Checksum().Equal(m1.Checksum()))
	assertGetEqual(assert, m1, k1, v1)
	m1e = m1.Edit()
	m1e.Set(k1, v2)
	assertGetEqual(assert, m1e, k1, v2)
	assertGetEqual(assert, m1, k1, v1)
	m2 := m1e.Map()
	assertGetEqual(assert, m2, k1, v2)
	assert.False(m2.Checksum().Equal(m1.Checksum()))

	m2e := m2.Edit()
	m2e.Remove(k1)
	assertGetError(assert, m2e, k1)
	assert.Error(m2e.Remove(k1))
	m2got := m2e.Map()
	assertGetError(assert, m2got, k1)
	assert.False(m2got.Checksum().Equal(m2.Checksum()))
	assert.True(m2got.Checksum().Equal(em.Checksum()), "got=%s, want=%s", m2got, em)
}
