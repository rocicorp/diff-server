package db

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"
)

func reloadDB(assert *assert.Assertions, dir string) (db *DB) {
	sp, err := spec.ForDatabase(dir)
	assert.NoError(err)

	db, err = Load(sp)
	assert.NoError(err)

	return db
}

func TestInitialBehavior(t *testing.T) {
	assert := assert.New(t)

	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)

	db, err := Load(sp)
	assert.NoError(err)

	assert.False(db.Has("foo"))
	buf := &bytes.Buffer{}
	ok, err := db.Get("foo", buf)
	assert.False(ok)
	assert.NoError(err)
	assert.Equal("", buf.String())

	r, err := db.GetCode()
	assert.Equal(types.Blob{}, r)
	assert.Error(err, "no code bundle is registered")

	_, changed, err := db.MakeTx("origin1", types.NewEmptyBlob(db.Noms()), "function1", types.NewList(db.db), datetime.Now())
	assert.False(changed)
	assert.NoError(err)
}

func TestCodeWriteNotPermanentUntilCommit(t *testing.T) {
	assert := assert.New(t)

	db, dir := LoadTempDB(assert)

	check := func(present bool) {
		b, err := db.GetCode()
		if present {
			assert.NotNil(b)
			assert.NoError(err)

			buf := &bytes.Buffer{}
			_, err = io.Copy(buf, b.Reader())
			assert.NoError(err)

			assert.Equal("code1", buf.String())
		} else {
			assert.Equal(types.Blob{}, b)
			assert.Error(err, "No code bundle is registered")
		}
	}

	check(false)

	err := db.PutCode(types.NewBlob(db.db, strings.NewReader("code1")))
	assert.NoError(err)

	check(true)

	db = reloadDB(assert, dir)

	check(false)
}

func TestDataWriteNotPermanentUntilCommit(t *testing.T) {
	assert := assert.New(t)

	db, dir := LoadTempDB(assert)

	check := func(present bool) {
		ok, err := db.Has("foo")
		assert.Equal(present, ok)
		assert.NoError(err)

		buf := &bytes.Buffer{}
		ok, err = db.Get("foo", buf)
		assert.Equal(present, ok)
		assert.NoError(err)

		if present {
			assert.Equal("42\n", buf.String())
		} else {
			assert.Equal("", buf.String())
		}
	}

	check(false)

	err := db.Put("foo", strings.NewReader("42"))
	assert.NoError(err)

	check(true)

	db = reloadDB(assert, dir)

	check(false)
}

func TestInitialCommitChangesCodeOnly(t *testing.T) {
	assert := assert.New(t)

	db, dir := LoadTempDB(assert)

	check := func(present bool) {
		b, err := db.GetCode()
		if present {
			assert.NotNil(b)
			assert.NoError(err)
			buf := &bytes.Buffer{}
			_, err = io.Copy(buf, b.Reader())
			assert.NoError(err)
			assert.Equal("code1", buf.String())
		} else {
			assert.Equal(types.Blob{}, b)
			assert.Error(err, "no code bundle is registered")
		}
	}

	check(false)

	err := db.PutCode(types.NewBlob(db.db, strings.NewReader("code1")))
	assert.NoError(err)

	check(true)

	tx, changed, err := db.MakeTx("origin1", types.NewEmptyBlob(db.db), "function1", types.NewList(db.db), datetime.Now())
	assert.True(changed)
	assert.NoError(err)

	_, err = db.Commit(tx)
	assert.NoError(err)

	check(true)

	ok, err := db.Get("foo", ioutil.Discard)
	assert.False(ok)
	assert.NoError(err)

	db = reloadDB(assert, dir)

	check(true)

	ok, err = db.Get("foo", ioutil.Discard)
	assert.False(ok)
	assert.NoError(err)
}

func TestDataBasics(t *testing.T) {
	assert := assert.New(t)

	db, dir := LoadTempDB(assert)

	err := db.Put("foo", strings.NewReader("42"))
	assert.NoError(err)

	tx, changed, err := db.MakeTx("origin1", types.NewEmptyBlob(db.db), "function1", types.NewList(db.db), datetime.Now())
	assert.True(changed)
	assert.NoError(err)

	_, err = db.Commit(tx)
	assert.NoError(err)

	ok, err := db.Has("foo")
	assert.True(ok)
	assert.NoError(err)

	buf := &bytes.Buffer{}
	ok, err = db.Get("foo", buf)
	assert.True(ok)
	assert.Nil(err)
	assert.Equal("42\n", buf.String())

	db = reloadDB(assert, dir)

	ok, err = db.Has("foo")
	assert.True(ok)
	assert.NoError(err)

	buf.Reset()
	ok, err = db.Get("foo", buf)
	assert.True(ok)
	assert.Nil(err)
	assert.Equal("42\n", buf.String())
}

func TestLoadHistorical(t *testing.T) {
	assert := assert.New(t)

	db, _ := LoadTempDB(assert)

	vals := []int{42, 43}
	refs := make([]types.Ref, len(vals))
	for i, v := range vals {
		err := db.Put("foo", strings.NewReader(fmt.Sprintf("%d", v)))
		assert.NoError(err)

		tx, changed, err := db.MakeTx("origin1", types.NewEmptyBlob(db.db), "function1", types.NewList(db.db), datetime.Now())
		assert.True(changed)
		assert.NoError(err, v)

		r, err := db.Commit(tx)
		assert.NoError(err, v)
		refs[i] = r
	}

	db, err := db.Fork(refs[0].TargetHash())
	assert.NoError(err)

	buf := &bytes.Buffer{}
	ok, err := db.Get("foo", buf)
	assert.True(ok)
	assert.Nil(err)
	assert.Equal("42\n", buf.String())

	err = db.Put("foo", strings.NewReader(fmt.Sprintf("%d", 44)))
	assert.NoError(err)

	c, changed, err := db.MakeTx("origin1", types.NewEmptyBlob(db.db), "function1", types.NewList(db.db), datetime.Now())
	assert.True(changed)

	_, err = db.Commit(c)
	assert.Error(err, "Dataset head is not ancestor of commit")
}

func TestCommitMarshal(t *testing.T) {
	assert := assert.New(t)

	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)

	db, err := Load(sp)
	assert.NoError(err)

	c := Commit{}
	c.Parents = append(c.Parents, types.NewRef(db.Noms().WriteValue(types.String("p1"))))
	c.Meta.Date = datetime.Now()
	c.Meta.Tx.Origin = "o1"
	c.Meta.Tx.Code = types.NewRef(db.Noms().WriteValue(types.NewBlob(db.Noms(), strings.NewReader("codecodecode"))))
	c.Meta.Tx.Name = "func1"
	c.Meta.Tx.Args = types.NewList(db.Noms(), types.String("foo"), types.Number(42))
	c.Value.Data = types.NewRef(types.NewMap(db.Noms(), types.String("foo"), types.Number(88)))
	c.Value.Code = types.NewRef(types.NewBlob(db.Noms(), strings.NewReader("other code")))

	v, err := marshal.Marshal(db.Noms(), c)
	assert.NoError(err)

	var actual Commit
	err = marshal.Unmarshal(v, &actual)
	assert.True(v.Equals(marshal.MustMarshal(db.Noms(), actual)))

	// TODO: test other forms
}
