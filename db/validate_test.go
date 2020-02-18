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

func TestValidate(t *testing.T) {
	assert := assert.New(t)

	db, dir := LoadTempDB(assert)
	fmt.Println(dir)
	db.PutBundle(types.NewBlob(db.Noms(), strings.NewReader("function log(k, v) { var val = db.get(k) || []; val.push(v); db.put(k, val); }")))
	noms := db.noms

	list := func(items ...string) types.List {
		r := types.NewList(noms).Edit()
		for i := 0; i < len(items); i++ {
			r.Append(types.String(items[i]))
		}
		return r.List()
	}

	epoch := datetime.DateTime{}
	g := makeGenesis(noms, "")
	d1 := types.NewMap(noms,
		types.String("foo"),
		list("bar"))
	d2 := types.NewMap(noms,
		types.String("foo"),
		list("bar", "baz"))

	tx1 := makeTx(
		noms,
		noms.WriteValue(g.Original),
		epoch,
		"log",
		list("foo", "bar"),
		noms.WriteValue(d1))
	noms.WriteValue(tx1.Original)

	tx1b := makeTx(
		noms,
		noms.WriteValue(g.Original),
		epoch,
		"log",
		list("foo", "bar"),
		noms.WriteValue(d2)) // incorrect, should be d1
	noms.WriteValue(tx1b.Original)

	tx2 := makeTx(
		noms,
		noms.WriteValue(tx1.Original),
		epoch,
		"log",
		list("foo", "baz"),
		noms.WriteValue(d2))
	noms.WriteValue(tx2.Original)

	tx2b := makeTx(
		noms,
		noms.WriteValue(tx1b.Original), // basis is incorrect
		epoch,
		"log",
		list("foo", "baz"),
		noms.WriteValue(
			d2.Edit().Set(
				types.String("foo"),
				list("bar", "baz", "baz")).Map()))
	noms.WriteValue(tx2b.Original)

	tx3 := makeTx(
		noms,
		noms.WriteValue(g.Original),
		epoch,
		"log",
		list("foo", "quux"),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("quux"))))
	noms.WriteValue(tx3.Original)
	ro1 := makeReorder(
		noms,
		noms.WriteValue(tx1.Original),
		epoch,
		noms.WriteValue(tx3.Original),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar", "quux"))))
	noms.WriteValue(ro1.Original)
	ro1b := makeReorder(
		noms,
		noms.WriteValue(tx1.Original),
		epoch,
		noms.WriteValue(tx3.Original),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar", "monkey")))) // incorrect
	noms.WriteValue(ro1b.Original)
	ro1c := makeReorder(
		noms,
		noms.WriteValue(tx1b.Original), // incorrect basis
		epoch,
		noms.WriteValue(tx3.Original),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar", "baz", "quux"))))
	noms.WriteValue(ro1c.Original)

	tc := []struct {
		in   Commit
		diff string
		err  string
	}{
		{tx1, "", ""},
		{tx1b, ".value {\n-   data: #atbvqcfprt13l5sadlpohu48tuctgmt4\n+   data: #a7u0iuqarbmjs9dnrf7d0fcotjrhdaaf\n  }\n", ""},
		{tx2, "", ""},
		{tx2b, "", ""}, // validate is non-recursive, so even though the basis is wrong, it doesn't notice.
		{tx3, "", ""},
		{ro1, "", ""},
		{ro1b, ".value {\n-   data: #1f3f1stoa9pit2jctse1svtl9vm01sbk\n+   data: #cdvf5afbdn7vpmj2ag7mhesrce5joob9\n  }\n", ""},
		{ro1c, "", ""}, // again, we don't see the incorrect basis
		{g, "", "Invalid commit type: CommitTypeGenesis"},
	}

	for i, t := range tc {
		label := fmt.Sprintf("test case %d", i)
		replayed, err := validate(db, t.in)
		db.noms.Flush()
		if t.err != "" {
			assert.EqualError(err, t.err, label)
			continue
		}
		assert.NoError(err, label)
		if t.diff == "" {
			assert.True(t.in.Original.Equals(replayed.Original), label+"\ndiff: "+diff.Diff(t.in.Original, replayed.Original))
		} else {
			assert.Equal(t.diff, diff.Diff(t.in.Original, replayed.Original), label)
		}
	}
}
