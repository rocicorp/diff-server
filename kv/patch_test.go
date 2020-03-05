package kv

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert" 
	"roci.dev/diff-server/util/noms/memstore"
)

func TestDiff(t *testing.T) {
	assert := assert.New(t)

	tc := []struct {
		label          string
		from           string
		to             string
		expectedResult []string
		expectedError  string
	}{
		{"insert",
			`map {}`, `map{"foo":"bar"}`, []string{`{"op":"add","path":"/foo","value":"bar"}`}, ""},
		{"remove",
			`map{"foo":"bar"}`, `map {}`, []string{`{"op":"remove","path":"/foo"}`}, ""},
		{"replace",
			`map{"foo":"bar"}`, `map {"foo":"baz"}`, []string{`{"op":"replace","path":"/foo","value":"baz"}`}, ""},
		{"escape-1",
			`map {}`, `map{"/":"foo"}`, []string{`{"op":"add","path":"/~1","value":"foo"}`}, ""},
		{"escape-2",
			`map {}`, `map{"~":"foo"}`, []string{`{"op":"add","path":"/~0","value":"foo"}`}, ""},
		{"deep",
			`map {"foo":map{"bar":"baz"}}`, `map {"foo":map{"bar":"quux"}}`,
			[]string{`{"op":"replace","path":"/foo","value":{"bar":"quux"}}`}, ""},
		{"all-types",
			`map{}`, `map {"foo":map{"b":true,"i":42,"f":88.8,"s":"monkey","a":[],"a2":[true,42,8.88E1],"o":map{}}}`,
			[]string{`{"op":"add","path":"/foo","value":{"a":[],"a2":[true,42,8.88E1],"b":true,"f":8.88E1,"i":42,"o":{},"s":"monkey"}}`}, ""},
		{"multiple",
			`map {"a":"a","b":"b"}`, `map {"b":"bb","c":"c"}`,
			[]string{
				`{"op":"remove","path":"/a"}`,
				`{"op":"replace","path":"/b","value":"bb"}`,
				`{"op":"add","path":"/c","value":"c"}`,
			}, ""},
	}

	noms := memstore.New()
	for _, t := range tc {
		fv := nomdl.MustParse(noms, t.from).(types.Map)
		fm := NewMapFromNoms(noms, fv)
		tv := nomdl.MustParse(noms, t.to).(types.Map)
		tm := NewMapFromNoms(noms, tv)
		r := []Operation{}
		r, err := Diff(fm, tm, r)
		if t.expectedError == "" {
			assert.NoError(err, t.label)
			j, err := json.Marshal(r)
			assert.NoError(err, t.label)
			assert.Equal("["+strings.Join(t.expectedResult, ",")+"]", string(j), t.label)
			got, err := ApplyPatch(fm, r)
			es, gots := types.EncodedValue(tm.nm), types.EncodedValue(got.nm)
			assert.Equal(es, gots, "%s expected %s got %s", t.label, es, gots)
			assert.True(tm.Checksum().Equal(got.Checksum()), "%s expected %s got %s", t.label, es, gots)
		} else {
			assert.EqualError(err, t.expectedError, t.label)
			// buf might have arbitrary data, not part of the contract
		}
	}
}

func TestTopLevelRemove(t *testing.T) {
	// Diff doesn't currently generate a top level remove, so test here.
	assert := assert.New(t)
	noms := memstore.New()

	fs, ts := `map {"a":"a","b":"b"}`, `map {"b":"bb"}`
	fv := nomdl.MustParse(noms, fs).(types.Map)
	fm := NewMapFromNoms(noms, fv)
	tv := nomdl.MustParse(noms, ts).(types.Map)
	tm := NewMapFromNoms(noms, tv)

	ops := []Operation{
		Operation{OpRemove, "/", []byte{}},
		Operation{OpReplace, "/b", []byte("\"bb\"")},
	}
	r, err := ApplyPatch(fm, ops)
	assert.NoError(err)
	assert.NoError(err)
	assert.Equal(types.EncodedValue(r.nm), types.EncodedValue(tm.nm))
	// TODO uncomment when canonicalization works.
	// assert.True(r.Checksum().Equal(tm.Checksum()))
}
