package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aboodman/replicant/util/noms/diff"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"
)

func TestRebase(t *testing.T) {
	assert := assert.New(t)

	db, dir := LoadTempDB(assert)
	fmt.Println(dir)
	noms := db.noms

	list := func(items ...string) types.List {
		r := types.NewList(noms).Edit()
		for i := 0; i < len(items); i++ {
			r.Append(types.String(items[i]))
		}
		return r.List()
	}

	data := func(items ...string) types.Map {
		r := types.NewMap(noms).Edit()
		r.Set(types.String("foo"), list(items...))
		return r.Map()
	}

	write := func(v types.Value) types.Ref {
		return noms.WriteValue(v)
	}

	assertEqual := func(c1, c2 Commit) {
		if c1.Original.Equals(c2.Original) {
			return
		}
		assert.Fail("Commits are unequal", "expected: %s, actual: %s, diff: %s", c1.Original.Hash(), c2.Original.Hash(), diff.Diff(c1.Original, c2.Original))
	}

	epoch := datetime.DateTime{}
	b := types.NewBlob(noms, strings.NewReader("function log(k, v) { var val = db.get(k) || []; val.push(v); db.put(k, val); }"))
	err := db.PutBundle(b)
	br := types.NewRef(b)
	assert.NoError(err)
	gCommit := db.head.Ref()

	// nop
	// onto: g - a
	// head: g - a - b
	// rslt: g - a - b
	_, err = db.Exec("log", list("foo", "bar"))
	assert.NoError(err)
	aCommit := db.head
	aCommitRef := aCommit.Ref()
	bCommit := makeTx(noms, db.head.Ref(), "test", epoch, br, "log", list("foo", "baz"), br, write(data("bar", "baz")))
	write(bCommit.Original)
	actual, err := rebase(db, db.head.Ref(), epoch, bCommit, types.Ref{})
	assert.NoError(err)
	assertEqual(bCommit, actual)

	// https://github.com/aboodman/replicant/issues/68
	// same as nop, except where there's also a 'local' branch whose head is > onto
	// local: g - a - b
	// onto:  g - a
	// head:  g - a - b
	// rslt:  g - a - b
	_, err = noms.SetHead(noms.GetDataset(local_dataset), bCommit.Ref())
	assert.NoError(err)
	db.Reload()
	actual, err = rebase(db, aCommit.Ref(), epoch, bCommit, types.Ref{})
	assert.NoError(err)
	assertEqual(bCommit, actual)

	// ff
	// onto: g - a - b
	// head: g - a
	// rslt: g - a - b
	_, err = noms.SetHead(noms.GetDataset(local_dataset), bCommit.Ref())
	assert.NoError(err)
	db.Reload()
	actual, err = rebase(db, db.head.Ref(), epoch, aCommit, types.Ref{})
	assert.Nil(err)
	assertEqual(bCommit, actual)

	// simple reorder
	// onto: g - a
	// head: g - b
	// rslt: g - a - ro(b)
	//         \ b /
	_, err = noms.SetHead(noms.GetDataset(local_dataset), aCommitRef)
	assert.NoError(err)
	db.Reload()
	bCommit = makeTx(noms, gCommit, "test", epoch, br, "log", list("foo", "baz"), br, write(data("baz")))
	expected := makeReorder(noms, db.head.Ref(), "test", epoch, write(bCommit.Original), br, write(data("bar", "baz")))
	noms.WriteValue(expected.Original)
	actual, err = rebase(db, db.head.Ref(), epoch, bCommit, types.Ref{})
	assert.NoError(err)
	assertEqual(expected, actual)

	// chained reorder
	// onto: g - a
	// head: g - b - c
	// rslt: g - a - ro(b) - ro(c)
	//         \ b /         /
	//            \ ------- c
	_, err = noms.SetHead(noms.GetDataset(local_dataset), aCommitRef)
	assert.NoError(err)
	db.Reload()
	cCommit := makeTx(noms, bCommit.Ref(), "test", epoch, br, "log", list("foo", "quux"), br, write(data("baz", "quux")))
	noms.WriteValue(cCommit.Original)
	bReorder := makeReorder(noms, db.head.Ref(), "test", epoch, write(bCommit.Original), br, write(data("bar", "baz")))
	noms.WriteValue(bReorder.Original)
	cReorder := makeReorder(noms, bReorder.Ref(), "test", epoch, cCommit.Ref(), br, write(data("bar", "baz", "quux")))
	noms.WriteValue(cReorder.Original)
	actual, err = rebase(db, db.head.Ref(), epoch, cCommit, types.Ref{})
	assert.NoError(err)
	assertEqual(cReorder, actual)

	// re-reorder
	// onto: g - a - b
	// head: g - a - ro(c)
	//         \ c /
	// rslt: g - a -  b  -  ro(ro(c))
	//         \    \ ro(c) /
	//          \  c  /
	_, err = noms.SetHead(noms.GetDataset(local_dataset), aCommitRef)
	assert.NoError(err)
	db.Reload()
	_, err = db.Exec("log", list("foo", "baz"))
	assert.NoError(err)
	bCommit = db.head
	cCommit = makeTx(noms, gCommit, "test", epoch, br, "log", list("foo", "quux"), br, write(data("quux")))
	noms.WriteValue(cCommit.Original)
	cReorder = makeReorder(noms, aCommit.Ref(), "test", epoch, cCommit.Ref(), br, write(data("bar", "quux")))
	noms.WriteValue(cReorder.Original)
	cReReorder := makeReorder(noms, bCommit.Ref(), "test", epoch, cReorder.Ref(), br, write(data("bar", "baz", "quux")))
	noms.WriteValue(cReReorder.Original)
	actual, err = rebase(db, bCommit.Ref(), epoch, cReorder, types.Ref{})
	assert.NoError(err)
	assertEqual(cReReorder, actual)

	// reject unsupported
	// head: g - a
	// onto: g ---- rj(b)
	//         \ b /
	// rslt: error: can't rebase reject commits
	_, err = noms.SetHead(noms.GetDataset(local_dataset), aCommitRef)
	assert.NoError(err)
	db.Reload()
	bCommit = makeTx(noms, gCommit, "test", epoch, br, "log", list("foo", "baz"), br, write(data("baz")))
	write(bCommit.Original)
	bRjCommit := makeReject(noms, gCommit, "test", epoch, bCommit.Ref(), "r1", br, write(data()))
	write(bRjCommit.Original)
	actual, err = rebase(db, aCommit.Ref(), epoch, bRjCommit, types.Ref{})
	assert.Error(err)
	assert.True(strings.HasPrefix(err.Error(), "Cannot rebase commit of type CommitTypeReject:"))
}
