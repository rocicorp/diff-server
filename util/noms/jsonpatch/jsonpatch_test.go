package jsonpatch

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/replicant/util/noms/memstore"
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
			`map{}`, `map {"foo":map{"b":true,"i":42,"f":88.8,"s":"monkey","a":[],"a2":[true,42,88.8],"o":map{}}}`,
			[]string{`{"op":"add","path":"/foo","value":{"a":[],"a2":[true,42,88.8],"b":true,"f":88.8,"i":42,"o":{},"s":"monkey"}}`}, ""},
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
		tv := nomdl.MustParse(noms, t.to).(types.Map)
		r := []Operation{}
		r, err := Diff(fv, tv, r)
		if t.expectedError == "" {
			assert.NoError(err, t.label)
			j, err := json.Marshal(r)
			assert.NoError(err)
			assert.Equal("["+strings.Join(t.expectedResult, ",")+"]", string(j), t.label)
			m, err := Apply(noms, fv, r)
			assert.Equal(types.EncodedValue(tv), types.EncodedValue(m), t.label)
		} else {
			assert.EqualError(err, t.expectedError, t.label)
			// buf might have arbitrary data, not part of the contract
		}
	}
}
