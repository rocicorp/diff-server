package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
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

	epoch := datetime.DateTime{}
	g := makeGenesis(noms)
	eb := types.NewEmptyBlob(noms)
	b1 := types.NewBlob(noms, strings.NewReader("function log(k, v) { var val = db.get(k) || []; val.push(v); db.put(k, val); }"))
	d1 := types.NewMap(noms,
		types.String("foo"),
		list("bar"))
	d2 := types.NewMap(noms,
		types.String("foo"),
		list("bar", "baz"))

	tx1 := makeTx(
		noms,
		noms.WriteValue(g.Original),
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		list("foo", "bar"),
		noms.WriteValue(eb),
		noms.WriteValue(d1))
	noms.WriteValue(tx1.Original)

	tx1b := makeTx(
		noms,
		noms.WriteValue(g.Original),
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		list("foo", "bar"),
		noms.WriteValue(eb),
		noms.WriteValue(d2)) // incorrect, should be d1
	noms.WriteValue(tx1b.Original)

	tx2 := makeTx(
		noms,
		noms.WriteValue(tx1.Original),
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		list("foo", "baz"),
		noms.WriteValue(eb),
		noms.WriteValue(d2))
	noms.WriteValue(tx2.Original)

	tx2b := makeTx(
		noms,
		noms.WriteValue(tx1b.Original), // basis is incorrect
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		list("foo", "baz"),
		noms.WriteValue(eb),
		noms.WriteValue(
			d2.Edit().Set(
				types.String("foo"),
				list("bar", "baz", "baz")).Map()))
	noms.WriteValue(tx2b.Original)

	tx3 := makeTx(
		noms,
		noms.WriteValue(g.Original),
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		list("foo", "quux"),
		noms.WriteValue(eb),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("quux"))))
	noms.WriteValue(tx3.Original)
	ro1 := makeReorder(
		noms,
		noms.WriteValue(tx1.Original),
		"o1",
		epoch,
		noms.WriteValue(tx3.Original),
		noms.WriteValue(eb),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar", "quux"))))
	noms.WriteValue(ro1.Original)
	ro1b := makeReorder(
		noms,
		noms.WriteValue(tx1.Original),
		"o1",
		epoch,
		noms.WriteValue(tx3.Original),
		noms.WriteValue(eb),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar", "monkey")))) // incorrect
	noms.WriteValue(ro1b.Original)
	ro1c := makeReorder(
		noms,
		noms.WriteValue(tx1b.Original), // incorrect basis
		"o1",
		epoch,
		noms.WriteValue(tx3.Original),
		noms.WriteValue(eb),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar", "quux"))))
	noms.WriteValue(ro1c.Original)

	rj1 := makeReject(
		noms,
		noms.WriteValue(tx1.Original),
		"o1",
		epoch,
		noms.WriteValue(tx3.Original),
		"reason1",
		noms.WriteValue(eb),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar"))))
	noms.WriteValue(rj1.Original)
	rj1b := makeReject(
		noms,
		noms.WriteValue(tx1.Original),
		"o1",
		epoch,
		noms.WriteValue(tx3.Original),
		"reason1",
		noms.WriteValue(eb),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar", "quux")))) // incorrect
	noms.WriteValue(rj1b.Original)
	rj1c := makeReject(
		noms,
		noms.WriteValue(tx1b.Original), // incorrect basis
		"o1",
		epoch,
		noms.WriteValue(tx3.Original),
		"reason1",
		noms.WriteValue(eb),
		noms.WriteValue(
			types.NewMap(noms, types.String("foo"), list("bar"))))
	noms.WriteValue(rj1c.Original)

	tc := []struct {
		in  Commit
		err string
	}{
		{tx1, ""},
		{tx1b, "Invalid commit ccn11ot718inmicflkuv3rueiqj02j2r: diff: .value {\n-   data: #atbvqcfprt13l5sadlpohu48tuctgmt4\n+   data: #a7u0iuqarbmjs9dnrf7d0fcotjrhdaaf\n  }\n"},
		{tx2, ""},
		{tx2b, "Invalid commit ccn11ot718inmicflkuv3rueiqj02j2r: diff: .value {\n-   data: #atbvqcfprt13l5sadlpohu48tuctgmt4\n+   data: #a7u0iuqarbmjs9dnrf7d0fcotjrhdaaf\n  }\n"},
		{tx3, ""},
		{ro1, ""},
		{ro1b, "Invalid commit ogrtfd10a6pa39qd18v1c9l4nl57kliu: diff: .value {\n-   data: #1f3f1stoa9pit2jctse1svtl9vm01sbk\n+   data: #cdvf5afbdn7vpmj2ag7mhesrce5joob9\n  }\n"},
		{ro1c, "Invalid commit ccn11ot718inmicflkuv3rueiqj02j2r: diff: .value {\n-   data: #atbvqcfprt13l5sadlpohu48tuctgmt4\n+   data: #a7u0iuqarbmjs9dnrf7d0fcotjrhdaaf\n  }\n"},
		{rj1, ""},
		{rj1b, "Invalid commit 9rrejcn1fo1lvo30biubb6e09hf56k7c: diff: .value {\n-   data: #cdvf5afbdn7vpmj2ag7mhesrce5joob9\n+   data: #a7u0iuqarbmjs9dnrf7d0fcotjrhdaaf\n  }\n"},
		{rj1c, "Invalid commit ccn11ot718inmicflkuv3rueiqj02j2r: diff: .value {\n-   data: #atbvqcfprt13l5sadlpohu48tuctgmt4\n+   data: #a7u0iuqarbmjs9dnrf7d0fcotjrhdaaf\n  }\n"},
	}

	for i, t := range tc {
		err := validate(db, t.in, types.Ref{})
		db.noms.Flush()
		if t.err == "" {
			assert.NoError(err, "test case %d", i)
		} else {
			assert.EqualError(err, t.err, "test case %d", i)
		}
	}
}
