package db

import (
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func TestEditorRoundtrip(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	db, err := Load(sp, "o1")
	assert.NoError(err)

	ed := &editor{noms: db.Noms(), data: db.head.Data(db.Noms()).Edit()}

	assert.False(ed.Has("foo"))
	v, err := ed.Get("foo")
	assert.NoError(err)
	assert.Nil(v)
	ok, err := ed.Del("foo")
	assert.NoError(err)
	assert.False(ok)
	assert.True(types.NewMap(db.Noms()).Equals(ed.Finalize()))

	assert.NoError(ed.Put("foo", types.Number(42)))
	ok, err = ed.Has("foo")
	assert.NoError(err)
	assert.True(ok)
	v, err = ed.Get("foo")
	assert.NoError(err)
	assert.True(v.Equals(types.Number(42)))
	assert.True(types.NewMap(db.Noms(), types.String("foo"), types.Number(42)).Equals(ed.Finalize()))

	ok, err = ed.Del("foo")
	assert.NoError(err)
	assert.True(ok)

	assert.False(ed.Has("foo"))
	v, err = ed.Get("foo")
	assert.NoError(err)
	assert.Nil(v)
	ok, err = ed.Del("foo")
	assert.NoError(err)
	assert.False(ok)
	assert.True(types.NewMap(db.Noms()).Equals(ed.Finalize()))
}
func TestEditorMutationAttempt(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	db, err := Load(sp, "o1")
	assert.NoError(err)

	var ed *editor
	tc := []struct {
		f        func()
		expected bool
	}{
		{func() {}, false},
		{func() { ed.Has("foo"); ed.Get("foo") }, false},
		{func() { ed.Del("foo") }, true},
		{func() { ed.Put("foo", types.String("bar")) }, true},
		{func() { ed.Put("foo", types.String("bar")); ed.Del("foo") }, true},
	}
	for i, t := range tc {
		ed = &editor{
			noms: db.Noms(),
			data: db.head.Data(db.Noms()).Edit(),
		}
		assert.False(ed.receivedMutAttempt, "test case %d: %#v", i, t.f)
		t.f()
		assert.Equal(t.expected, ed.receivedMutAttempt, "test case %d", i)
	}
}
