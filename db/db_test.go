package db

import (
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func reloadDB(assert *assert.Assertions, dir string) (db *DB) {
	sp, err := spec.ForDatabase(dir)
	assert.NoError(err)

	db, err = Load(sp, "test")
	assert.NoError(err)

	return db
}

func TestGenesis(t *testing.T) {
	assert := assert.New(t)

	db, _ := LoadTempDB(assert)

	assert.False(db.Has("foo"))
	b, err := db.Bundle()
	assert.NoError(err)
	assert.False(b == types.Blob{})
	assert.Equal(uint64(0), b.Len())

	assert.True(db.head.Original.Equals(makeGenesis(db.noms).Original))
}

func TestData(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)

	exp := types.String("bar")
	err := db.Put("foo", exp)
	assert.NoError(err)

	dbs := []*DB{
		db, reloadDB(assert, dir),
	}

	for _, d := range dbs {
		ok, err := d.Has("foo")
		assert.NoError(err)
		assert.True(ok)
		act, err := d.Get("foo")
		assert.NoError(err)
		assert.True(act.Equals(exp))

		ok, err = d.Has("bar")
		assert.NoError(err)
		assert.False(ok)

		act, err = d.Get("bar")
		assert.NoError(err)
		assert.Nil(act)
	}
}

func TestBundle(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)

	exp := types.NewBlob(db.noms, strings.NewReader("bundlebundle"))

	err := db.PutBundle(exp)
	assert.NoError(err)

	dbs := []*DB{db, reloadDB(assert, dir)}
	for _, d := range dbs {
		act, err := d.Bundle()
		assert.NoError(err)
		assert.True(exp.Equals(act))
	}
}

func TestExec(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)

	code := `function append(k, s) {
	var val = db.get(k) || [];
	val.push(s);
	db.put(k, val);
}
`

	db.PutBundle(types.NewBlob(db.noms, strings.NewReader(code)))

	out, err := db.Exec("append", types.NewList(db.noms, types.String("log"), types.String("foo")))
	assert.NoError(err)
	assert.Nil(out)
	out, err = db.Exec("append", types.NewList(db.noms, types.String("log"), types.String("bar")))
	assert.NoError(err)
	assert.Nil(out)

	dbs := []*DB{db, reloadDB(assert, dir)}
	for _, d := range dbs {
		act, err := d.Get("log")
		assert.NoError(err)
		assert.True(types.NewList(d.noms, types.String("foo"), types.String("bar")).Equals(act))
	}
}

func TestReadTransaction(t *testing.T) {
	assert := assert.New(t)
	db, _ := LoadTempDB(assert)

	code := `function write(v) { db.put("foo", v) } function read() { return db.get("foo") }`

	db.PutBundle(types.NewBlob(db.noms, strings.NewReader(code)))

	out, err := db.Exec("write", types.NewList(db.noms, types.String("bar")))
	assert.NoError(err)
	assert.Nil(out)
	h := db.head.Original

	out, err = db.Exec("read", types.NewList(db.noms))
	assert.NoError(err)
	assert.Equal("bar", string(out.(types.String)))

	// Read-only transactions shouldn't add a commit
	assert.True(h.Equals(db.head.Original))
}
