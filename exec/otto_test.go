package exec

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	jsnoms "github.com/aboodman/replicant/util/noms/json"
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

func (d db) Scan(opts ScanOptions) (r []ScanItem, err error) {
	lim := opts.Limit
	if lim == 0 {
		lim = math.MaxInt32
	}
	for k, v := range d.data {
		if k < opts.StartAtID {
			continue
		}
		if opts.StartAfterID != "" && k <= opts.StartAfterID {
			continue
		}
		if !strings.HasPrefix(k, opts.Prefix) {
			continue
		}
		if len(r) == lim {
			continue
		}
		r = append(r, ScanItem{
			ID:    k,
			Value: jsnoms.Make(d.noms, v),
		})
	}
	sort.Slice(r, func(i, j int) bool {
		return r[i].ID < r[j].ID
	})
	return r, nil
}

func (d db) Del(id string) (ok bool, err error) {
	if _, ok := d.data[id]; ok {
		delete(d.data, id)
		return ok, nil
	}
	return false, nil
}

func TestArgsToJSToStorage(t *testing.T) {
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

func TestPutHasGetRoundtrip(t *testing.T) {
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
	db.put("hot", "dog");
	assert(db.has("foo"));
	assert("bar" === db.get("foo"));
	var results = db.scan({
		fromID: "f",
	});
	assert(results.length == 2);
	assert(results[0].id == "foo");
	assert(results[0].value == "bar");
	assert(results[1].id == "hot");
	assert(results[01].value == "dog");
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

func TestErrors(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	noms := sp.GetDatabase()
	d := db{noms, map[string]types.Value{}}
	args := types.NewList(d.noms)

	tc := []struct {
		bundle string
		fn     string
		err    string
	}{
		{"!!not valid javascript!!!", "", "bundle.js: Line 1:7 Unexpected identifier (and 2 more errors)"},
		{"throw new Error('bonk')", "", "Error: bonk\n    at bundle.js:1:11\n"},
		{"function bonk() { throw new Error('bonk'); }", "bonk", "Error: bonk\n    at bonk (bundle.js:1:29)\n    at apply (<native code>)\n    at recv (bootstrap.js:64:12)\n"},
		{"", "bonk", "Unknown function: bonk"},
	}

	for i, t := range tc {
		msg := fmt.Sprintf("test case %d,: code: %s, fn: %s", i, t.bundle, t.fn)
		out, err := Run(d, strings.NewReader(t.bundle), t.fn, args)
		assert.EqualError(err, t.err, msg)
		assert.Nil(out, msg)
	}
}

func TestDelete(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	noms := sp.GetDatabase()
	d := db{noms, map[string]types.Value{}}
	d.data["foo"] = types.String("bar")
	assert.NotEmpty(d.data)

	bundle := "function del(id) { return db.del(id); }"
	result, err := Run(d, strings.NewReader(bundle), "del", types.NewList(d.noms, types.String("foo")))
	assert.NoError(err)
	assert.Empty(d.data)
	assert.True(types.Bool(true).Equals(result))

	result, err = Run(d, strings.NewReader(bundle), "del", types.NewList(d.noms, types.String("foo")))
	assert.NoError(err)
	assert.Empty(d.data)
	assert.True(types.Bool(false).Equals(result))
}

// TODO : so much more to test
// - putting invalid / non-jsonable data / non-nomsable
// - returning invalid / non-jsonable data / non-nomsable
// - scripts that run forever

// Then eventually (not implemented yet)
// - determinism
// - sandboxing
