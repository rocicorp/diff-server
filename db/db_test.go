package db

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func reloadDB(assert *assert.Assertions, dir string) (db *DB) {
	sp, err := spec.ForDatabase(dir)
	assert.NoError(err)

	db, err = Load(sp)
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

	assert.True(db.head.Original.Equals(makeGenesis(db.noms, "").Original))
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
	db, err := Load(sp)
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

	act, err := db.Bundle()
	assert.NoError(err)
	assert.True(exp.Equals(act))

	db = reloadDB(assert, dir)
	act, err = db.Bundle()
	assert.NoError(err)
	assert.True(act.Empty())
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

func TestExecBatch(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)

	code := `function append(k, s) {
	var val = db.get(k) || [];
	val.push(s);
	db.put(k, val);
	return s;
}
`

	db.PutBundle(types.NewBlob(db.noms, strings.NewReader(code)))

	batch := []BatchItem{
		BatchItem{"append", types.NewList(db.noms, types.String("log"), types.String("foo"))},
		BatchItem{"append", types.NewList(db.noms, types.String("log"), types.String("bar"))},
	}

	out, be, err := db.ExecBatch(batch)
	assert.NoError(err)
	assert.Nil(be)
	assert.Equal(2, len(out))
	assert.Equal("foo", string(out[0].Result.(types.String)))
	assert.Equal("bar", string(out[1].Result.(types.String)))

	dbs := []*DB{db, reloadDB(assert, dir)}
	for _, d := range dbs {
		act, err := d.Get("log")
		assert.NoError(err)
		assert.True(types.NewList(d.noms, types.String("foo"), types.String("bar")).Equals(act))
	}
}

func TestExecBatchError(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)

	code := `function append(k, s) {
	if (s == 'bar') {
		throw new Error('error');
	}
	var val = db.get(k) || [];
	val.push(s);
	db.put(k, val);
	return s;
}
`

	db.PutBundle(types.NewBlob(db.noms, strings.NewReader(code)))

	batch := []BatchItem{
		BatchItem{"append", types.NewList(db.noms, types.String("log"), types.String("foo"))},
		BatchItem{"append", types.NewList(db.noms, types.String("log"), types.String("bar"))},
	}

	out, be, err := db.ExecBatch(batch)
	assert.NoError(err)
	assert.Equal(1, be.Index)
	assert.EqualError(be, "Error: error\n    at append (bundle.js:3:13)\n    at apply (<native code>)\n    at recv (bootstrap.js:67:12)\n")
	assert.Nil(out)

	dbs := []*DB{db, reloadDB(assert, dir)}
	for _, d := range dbs {
		act, err := d.Get("log")
		assert.NoError(err)
		assert.Nil(act) // commit doesn't happen in the case of error
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

func TestLoadBadSpec(t *testing.T) {
	assert := assert.New(t)

	sp, err := spec.ForDatabase("http://localhost:6666") // not running, presumably
	assert.NoError(err)
	db, err := Load(sp)
	assert.Nil(db)
	assert.EqualError(err, "Get http://localhost:6666/root/: dial tcp [::1]:6666: connect: connection refused")

	srv := httptest.NewServer(http.NotFoundHandler())
	sp, err = spec.ForDatabase(srv.URL)
	assert.NoError(err)
	db, err = Load(sp)
	assert.Nil(db)
	assert.EqualError(err, "Unexpected response: Not Found: 404 page not found")
}
