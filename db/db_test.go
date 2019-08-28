package db

import (
	"fmt"
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

func TestDel(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	db, err := Load(sp, "test")
	assert.NoError(err)

	err = db.Put("foo", types.String("bar"))
	assert.NoError(err)

	ok, err := db.Has("foo")
	assert.NoError(err)
	assert.True(ok)

	ok, err = db.Del("foo")
	assert.NoError(err)
	assert.True(ok)

	ok, err = db.Has("foo")
	assert.NoError(err)
	assert.False(ok)

	ok, err = db.Del("foo")
	assert.NoError(err)
	assert.False(ok)
}

func TestBundleInvalid(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)

	err := db.PutBundle(types.NewBlob(db.noms, strings.NewReader("bundlebundle")))
	assert.EqualError(err, "ReferenceError: 'bundlebundle' is not defined\n    at bundle.js:1:1\n")

	dbs := []*DB{db, reloadDB(assert, dir)}
	for _, d := range dbs {
		act, err := d.Bundle()
		assert.NoError(err)
		assert.True(types.NewEmptyBlob(db.noms).Equals(act))
	}
}

func TestBundleUnversioned(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)

	exp := types.NewBlob(db.noms, strings.NewReader("function foo(){}"))
	err := db.PutBundle(exp)
	assert.NoError(err)

	dbs := []*DB{db, reloadDB(assert, dir)}
	for _, d := range dbs {
		act, err := d.Bundle()
		assert.NoError(err)
		assert.True(exp.Equals(act))
	}
}

func TestUpgrade(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)
	fmt.Println(dir)

	tc := []struct {
		nb            string
		expectedError string
		expectUpgrade bool
	}{
		{"", "", false},                 // bundle is unchanged from default
		{"function foo(){}", "", true},  // unversioned upgrade
		{"function foo(){}", "", false}, // unchanged
		{"function bar(){}", "", true},  // unversioned upgrade
		{"function codeVersion() { return 'bonk'; }", "codeVersion() must return a number", false}, // invalid impl of codeVersion()
		{"function codeVersion() { return 0.1; }", "", true},                                       // unversioned->versioned upgrade
		{"function codeVersion() { return 0.1; }", "", false},                                      // unchanged
		{"function codeVersion() { return 1.1; }", "", true},                                       // versioned upgrade
		{"function codeVersion() { return 0.5; }", "", false},                                      // downgrade
	}

	for i, t := range tc {
		msg := fmt.Sprintf("test case %d (%s)", i, t.nb)
		prevHead := db.head.Original

		proposed := types.NewBlob(db.noms, strings.NewReader(t.nb))
		err := db.PutBundle(proposed)
		if t.expectedError != "" {
			assert.EqualError(err, t.expectedError)
		} else {
			assert.NoError(err, msg)
		}

		currBundle, err := db.Bundle()
		assert.NoError(err, msg)

		if t.expectUpgrade {
			assert.False(db.head.Original.Equals(prevHead))
			assert.True(proposed.Equals(currBundle), msg)
		} else {
			assert.True(db.head.Original.Equals(prevHead))
		}
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
