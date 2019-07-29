package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aboodman/replicant/util/noms/reachable"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	assert := assert.New(t)

	db, dir := LoadTempDB(assert)
	fmt.Println(dir)
	noms := db.noms

	epoch := datetime.DateTime{}
	g := makeGenesis(noms)
	eb := types.NewEmptyBlob(noms)
	b1 := types.NewBlob(noms, strings.NewReader("function log(k, v) { var val = db.get(k) || []; val.push(v); db.put(k, val); }"))
	d1 := types.NewMap(noms,
		types.String("foo"),
		types.NewList(noms, types.String("bar")))
	d2 := types.NewMap(noms,
		types.String("foo"),
		types.NewList(noms, types.String("bar"), types.String("baz")))

	tx1 := makeTx(
		noms,
		noms.WriteValue(g.Original),
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		types.NewList(noms,
			types.String("foo"),
			types.String("bar")),
		types.NewRef(eb),
		noms.WriteValue(d1))
	noms.WriteValue(tx1.Original)

	tx1b := makeTx(
		noms,
		noms.WriteValue(g.Original),
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		types.NewList(noms,
			types.String("foo"),
			types.String("bar")),
		types.NewRef(eb),
		noms.WriteValue(d2)) // incorrect, should be d1
	noms.WriteValue(tx1b.Original)

	tx2 := makeTx(
		noms,
		noms.WriteValue(tx1.Original),
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		types.NewList(noms,
			types.String("foo"),
			types.String("baz")),
		types.NewRef(eb),
		noms.WriteValue(d2))
	noms.WriteValue(tx2.Original)

	tx2b := makeTx(
		noms,
		noms.WriteValue(tx1b.Original), // basis is incorrect
		"o1",
		epoch,
		noms.WriteValue(b1),
		"log",
		types.NewList(noms,
			types.String("foo"),
			types.String("baz")),
		types.NewRef(eb),
		noms.WriteValue(
			d2.Edit().Set(
				types.String("foo"),
				types.NewList(noms, types.String("bar"), types.String("baz"), types.String("baz"))).Map()))
	noms.WriteValue(tx2b.Original)

	tc := []struct {
		in  Commit
		err string
	}{
		{tx1, ""},
		{tx1b, "Invalid commit ccn11ot718inmicflkuv3rueiqj02j2r: diff: .value {\n-   data: #atbvqcfprt13l5sadlpohu48tuctgmt4\n+   data: #a7u0iuqarbmjs9dnrf7d0fcotjrhdaaf\n  }\n"},
		{tx2, ""},
		{tx2b, "Invalid commit ccn11ot718inmicflkuv3rueiqj02j2r: diff: .value {\n-   data: #atbvqcfprt13l5sadlpohu48tuctgmt4\n+   data: #a7u0iuqarbmjs9dnrf7d0fcotjrhdaaf\n  }\n"},
	}

	for i, t := range tc {
		err := validate(db, reachable.New(noms), t.in)
		db.noms.Flush()
		if t.err == "" {
			assert.NoError(err, "test case %d", i)
		} else {
			assert.EqualError(err, t.err, "test case %d", i)
		}
	}

	// cases:
	// tx
	// - true one with parents
	// - false one false because of self
	// - false because of parents
	//
	// reorder
	// - true one
	// - false because of self
	// - false because of parents
	//
	//
}
