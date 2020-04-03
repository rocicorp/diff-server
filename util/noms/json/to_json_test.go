// Copyright 2019 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package json

import (
	"bytes"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/suite"
)

func TestToJSONSuite(t *testing.T) {
	suite.Run(t, &ToJSONSuite{})
}

type ToJSONSuite struct {
	suite.Suite
	vs *types.ValueStore
}

func (suite *ToJSONSuite) SetupTest() {
	st := &chunks.TestStorage{}
	suite.vs = types.NewValueStore(st.NewView())
}

func (suite *ToJSONSuite) TearDownTest() {
	suite.vs.Close()
}

func (suite *ToJSONSuite) TestToJSON() {
	tc := []struct {
		desc     string
		in       types.Value
		exp      string
		expError string
	}{
		{"null", Null(), `null`, ""},
		{"true", types.Bool(true), "true", ""},
		{"false", types.Bool(false), "false", ""},
		{"42", types.Number(42), "42", ""},
		{"88.8", types.Number(88.8), "8.88E1", ""},
		{"empty string", types.String(""), `""`, ""},
		{"foobar", types.String("foobar"), `"foobar"`, ""},
		{"strings with newlines", types.String(`"\nmonkey`), `"\"\\nmonkey"`, ""},
		{"unnamed struct", types.NewStruct("", types.StructData{}), "", "Unsupported struct type: Struct {}"},
		{"named struct", types.NewStruct("Person", types.StructData{}), "", "Unsupported struct type: Struct Person {}"},
		{"bad null struct", types.NewStruct("Null", types.StructData{"foo": types.String("bar")}), "", "Unsupported struct type: Struct Null {\n  foo: String,\n}"},
		{"empty list", types.NewList(suite.vs), "[]", ""},
		{"non-empty list", types.NewList(suite.vs, types.Number(42), types.String("foo")), `[42,"foo"]`, ""},
		{"sets", types.NewSet(suite.vs), "", "Unsupported kind: Set"},
		{"map non-string key", types.NewMap(suite.vs, types.Number(42), types.Number(42)), "", "Map key kind Number not supported"},
		{"empty map", types.NewMap(suite.vs), "{}", ""},
		{"non-empty map", types.NewMap(suite.vs, types.String("foo"), types.String("bar"), types.String("baz"), types.Number(42)), `{"baz":42,"foo":"bar"}`, ""},
		{"map with newlines in strings", types.NewMap(suite.vs, types.String("foo\n"), types.String("ba\nr")), `{"foo\n":"ba\nr"}`, ""},
		{"complex value", types.NewMap(suite.vs,
			types.String("list"), types.NewList(suite.vs,
				types.NewMap(suite.vs,
					types.String("foo"), types.String("bar"),
					types.String("hot"), types.Number(42),
					types.String("null"), Null()))), `{"list":[{"foo":"bar","hot":42,"null":null}]}`, ""},
	}

	for _, t := range tc {
		buf := &bytes.Buffer{}
		err := ToJSON(t.in, buf)
		if t.expError != "" {
			suite.EqualError(err, t.expError, t.desc)
			suite.Equal("", string(buf.Bytes()), t.desc)
		} else {
			suite.NoError(err)
			suite.Equal(t.exp, string(buf.Bytes()), t.desc)
		}
	}
}
