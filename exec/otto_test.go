package exec

import (
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

type db struct {
	noms types.ValueReadWriter
	data map[string]types.Value
}

func (d db) Noms() types.ValueReadWriter {
	return d.noms
}

func (d db) Put(id string, val types.Value) error {
	d.data[id] = val
	return nil
}

func (d db) Has(id string) (ok bool, err error) {
	_, ok = d.data[id]
	return
}

func (d db) Get(id string) (v types.Value, err error) {
	v = d.data[id]
	return
}

func TestPut(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	noms := sp.GetDatabase()
	d := db{noms, map[string]types.Value{}}
	code := `function put(v) {db.put("foo", v)}`
	tc := []struct {
		in  types.Value
		err string
	}{
		{types.Bool(true), ""},
		{types.Bool(false), ""},
		{types.Number(0), ""},
		{types.Number(42), ""},
		{types.Number(88.8), ""},
		{types.Number(-1), ""},
		{types.String(""), ""},
		{types.String("bar"), ""},
		{types.NewList(noms), ""},
		{types.NewList(noms, types.String("foo"), types.String("bar")), ""},
		{types.NewList(noms, types.NewList(noms)), ""},
		{types.NewMap(noms, types.String("foo"), types.String("bar")), ""},
		{types.NewMap(noms, types.String("foo"), types.NewMap(noms, types.String("bar"), types.String("baz"))), ""},
	}

	for i, t := range tc {
		out, err := Run(d, strings.NewReader(code), "put", types.NewList(noms, t.in))
		assert.NoError(err, "test case %d", i)
		assert.Nil(out, "test case %d", i)
		assert.True(t.in.Equals(d.data["foo"]), "test case %d", i)
	}
}

func TestRoundtrip(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	noms := sp.GetDatabase()
	d := db{noms, map[string]types.Value{}}
	code := `
function assert(cond) {
	if (!cond) {
		throw new Error("unexpected condition");
	}
}

function test() {
	assert(!db.has("foo"));
	db.put("foo", "bar");
	assert(db.has("foo"));
	assert("bar" === db.get("foo"));
	return "hi";
}
`
	out, err := Run(d, strings.NewReader(code), "test", types.NewList(noms))
	assert.NoError(err)
	assert.True(types.String("hi").Equals(out))
}

func TestOutput(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	noms := sp.GetDatabase()
	d := db{noms, map[string]types.Value{}}
	args := types.NewList(noms,
		types.String("foo"), types.Number(42), types.NewList(noms,
			types.Bool(true), types.NewMap(noms,
				types.String("foo"), types.String("bar"))))
	out, err := Run(d, strings.NewReader("function echo(v1, v2, v3) { return [v1,v2,v3]}"), "echo", args)
	assert.NoError(err)
	assert.NotNil(out)
	assert.True(args.Equals(out), types.EncodedValue(out))

	out, err = Run(d, strings.NewReader("function add(a, b) { return a+b}"), "add", types.NewList(noms, types.Number(2), types.Number(3)))
	assert.NoError(err)
	assert.NotNil(out)
	assert.True(types.Number(5).Equals(out), types.EncodedValue(out))

	out, err = Run(d, strings.NewReader("function noOutput(a, b) {}"), "noOutput", types.NewList(noms))
	assert.NoError(err)
	assert.Nil(out)
}

// TODO : so much more to test
// - scripts that don't parse
// - functions that throw errors
// - putting invalid / non-jsonable data / non-nomsable
// - returning invalid / non-jsonable data / non-nomsable
// - scripts that run forever

// Then eventually (not implemented yet)
// - determinism
// - sandboxing
