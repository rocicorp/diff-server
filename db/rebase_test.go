package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"

	"roci.dev/replicant/util/noms/diff"
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

	write := func(v types.Value) types.Ref {
		return noms.WriteValue(v)
	}

	data := func(ds string) types.Map {
		if ds == "" {
			return types.NewMap(noms)
		}
		return types.NewMap(noms, types.String("foo"), list(strings.Split(ds, ",")...))
	}

	assertEqual := func(c1, c2 Commit) {
		if c1.Original.Equals(c2.Original) {
			return
		}
		fmt.Println(c1.Original.Hash(), c2.Original.Hash())
		assert.Fail("Commits are unequal", "expected: %s, actual: %s, diff: %s", c1.Original.Hash(), c2.Original.Hash(), diff.Diff(c1.Original, c2.Original))
	}

	g := db.head
	epoch := datetime.DateTime{}
	bundle := types.NewBlob(noms, strings.NewReader("function log(k, v) { var val = db.get(k) || []; val.push(v); db.put(k, val); }"))
	err := db.PutBundle(bundle)
	assert.NoError(err)
	bundleRef := types.NewRef(bundle)

	tx := func(basis Commit, arg string, ds string) Commit {
		d := data(ds)
		r := makeTx(
			noms,
			basis.Ref(),
			"test", // origin
			epoch,
			bundleRef,        // bundle
			"log",            // function
			list("foo", arg), // args
			basis.Value.Code, // result bundle
			write(d))         // result data
		write(r.Original)
		return r
	}

	ro := func(basis, subject Commit, ds string) Commit {
		d := data(ds)
		r := makeReorder(
			noms,
			basis.Ref(),
			"test",
			epoch,
			subject.Ref(),
			basis.Value.Code,
			write(d)) // result data
		write(r.Original)
		return r
	}

	rj := func(basis, subject, expected Commit, ds string) Commit {
		d := data(ds)
		r := makeReject(
			noms,
			basis.Ref(),
			"test",
			epoch,
			subject.Ref(),
			expected.Ref(),
			"",
			basis.Value.Code,
			write(d)) // result data
		write(r.Original)
		return r
	}

	test := func(onto, head, expected Commit, expectedError string) {
		noms.Flush()
		actual, err := rebase(db, onto.Ref(), epoch, head, types.Ref{})
		if expectedError != "" {
			assert.EqualError(err, expectedError)
			return
		}
		assert.NoError(err)
		write(actual.Original)
		noms.Flush()
		assertEqual(expected, actual)
	}

	// dest ff
	// onto: g
	// head: g - a
	// rslt: g - a
	(func() {
		a := tx(g, "a", "a")
		test(g, a, a, "")
	})()

	// https://github.com/aboodman/replicant/issues/68
	// same as dest ff, except where there's also a 'local' branch whose head is > onto
	// local: g - a
	// onto:  g
	// head:  g - a
	// rslt:  g - a
	(func() {
		a := tx(g, "a", "a")
		_, err := noms.SetHead(noms.GetDataset(LOCAL_DATASET), a.Ref())
		assert.NoError(err)
		db.Reload()
		test(g, a, a, "")
	})()

	// source ff
	// onto: g - a
	// head: g
	// rslt: g - a
	(func() {
		a := tx(g, "a", "a")
		test(a, g, a, "")
	})()

	// simple reorder
	// onto: g - a
	// head: g - b
	// rslt: g - a - ro(b)
	//         \ b /
	(func() {
		a := tx(g, "a", "a")
		b := tx(g, "b", "b")
		expected := ro(a, b, "a,b")
		test(a, b, expected, "")
	})()

	// chained reorder
	// onto: g - a
	// head: g - b - c
	// rslt: g - a - ro(b) - ro(c)
	//         \ b /         /
	//            \ ------- c
	(func() {
		a := tx(g, "a", "a")
		b := tx(g, "b", "b")
		c := tx(b, "c", "b,c")
		rob := ro(a, b, "a,b")
		roc := ro(rob, c, "a,b,c")
		test(a, c, roc, "")
	})()

	// re-reorder
	// onto: g - a - b
	// head: g - a - ro(c)
	//         \ c /
	// rslt: g - a -  b  -  ro(ro(c))
	//         \    \ ro(c) /
	//          \  c  /
	(func() {
		a := tx(g, "a", "a")
		b := tx(a, "b", "a,b")
		c := tx(g, "c", "c")
		roc := ro(a, c, "a,c")
		expected := ro(b, roc, "a,b,c")
		test(b, roc, expected, "")
	})()

	// reject unsupported
	// onto: g - a
	// head: g --- rj(b)
	//         \ b /
	// rslt: error: can't rebase reject commits
	(func() {
		a := tx(g, "a", "a")
		b := tx(g, "b", "b")
		rjb := rj(g, b, a, "")
		test(a, rjb, Commit{}, "Invalid commit type: CommitTypeReject")
	})()

	// nondeterm/ff
	// a client syncs a ff that is incorrect
	// onto: g
	// head: g - x
	// rslt: g ---- rj(x)
	//         \ x /
	(func() {
		x := tx(g, "x", "a")
		expected := tx(g, "x", "x")
		rjx := rj(g, x, expected, "") // commit is not applied, therefore data goes back to basis value, which is empty
		test(g, x, rjx, "")
	})()

	// nondeterm/ff2
	// a client syncs a ff that has an incorrect commit followed by a correct one
	// onto: g
	// head: g - x - b
	// rslt: g ---- rj(x) - ro(b)
	//         \ x / - b - /
	(func() {
		x := tx(g, "x", "a")
		b := tx(x, "b", "a,b")
		expectedX := tx(g, "x", "x")
		rjx := rj(g, x, expectedX, "")
		rob := ro(rjx, b, "b")
		test(g, b, rob, "")
	})()

	// nondeterm/ff3
	// a client syncs a ff that contains two incorrect commits in a row
	// onto: g
	// head: g - x - y
	// rslt: g ---- rj(x) - rj(y)
	//         \ x / - y - /
	(func() {
		x := tx(g, "x", "a")
		y := tx(x, "y", "a,b")
		expectedX := tx(g, "x", "x")
		rjx := rj(g, x, expectedX, "")
		expectedY := tx(x, "y", "a,y")
		rjy := rj(rjx, y, expectedY, "")
		test(g, y, rjy, "")
	})()

	// nondeterm/reorder
	// a client submits an incorrect commit that also needs to be reordered
	// we never get to the reordering because the commit fails validation
	// onto: g - a
	// head: g - x
	// rslt: g - a - rj(x)
	//         \ x /
	(func() {
		a := tx(g, "a", "a")
		x := tx(g, "x", "b")
		expected := tx(g, "x", "x")
		rjx := rj(a, x, expected, "a")
		test(a, x, rjx, "")
	})()

	// nondeterm/reorder2
	// a client syncs a ff that contains a reorder that has the wrong result
	// onto: g
	// head: g - a - xro(b)
	//         \ b /
	// rslt: g - a \ ------ rj(xro(b))
	//         \ b - xro(b) /
	(func() {
		a := tx(g, "a", "a")
		b := tx(g, "b", "b")
		xrob := ro(a, b, "a,x") // data is wrong, should be a,b
		expected := ro(a, b, "a,b")
		rjxrob := rj(a, xrob, expected, "a")
		test(g, xrob, rjxrob, "")
	})()
}
