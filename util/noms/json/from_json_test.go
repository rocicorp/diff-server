// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package json

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestLibTestSuite(t *testing.T) {
	suite.Run(t, &LibTestSuite{})
}

type LibTestSuite struct {
	suite.Suite
	vs *types.ValueStore
}

func (suite *LibTestSuite) SetupTest() {
	st := &chunks.TestStorage{}
	suite.vs = types.NewValueStore(st.NewView())
}

func (suite *LibTestSuite) TearDownTest() {
	suite.vs.Close()
}

func (suite *LibTestSuite) TestPrimitiveTypes() {
	vs := suite.vs
	suite.EqualValues(types.String("expected"), NomsValueFromDecodedJSON(vs, "expected"))
	suite.EqualValues(types.Bool(false), NomsValueFromDecodedJSON(vs, false))
	suite.EqualValues(types.Number(1.7), NomsValueFromDecodedJSON(vs, 1.7))
	suite.True(NomsValueFromDecodedJSON(vs, nil).Equals(Null()))
	suite.False(NomsValueFromDecodedJSON(vs, 1.7).Equals(types.Bool(true)))
}

func (suite *LibTestSuite) TestCompositeTypes() {
	vs := suite.vs

	// [false true null]
	suite.True(
		types.NewList(vs).Edit().Append(types.Bool(false)).Append(types.Bool(true)).Append(Null()).List().Equals(
			NomsValueFromDecodedJSON(vs, []interface{}{false, true, nil})))

	// [[false true null]]
	suite.True(
		types.NewList(vs).Edit().Append(
			types.NewList(vs).Edit().Append(types.Bool(false)).Append(types.Bool(true)).Append(Null()).List()).List().Equals(
			NomsValueFromDecodedJSON(vs, []interface{}{[]interface{}{false, true, nil}})))

	// {"string": "string",
	//  "list": [false true],
	//  "map": {"nested": "string"}
	// }
	m := types.NewMap(
		vs,
		types.String("string"),
		types.String("string"),
		types.String("list"),
		types.NewList(vs).Edit().Append(types.Bool(false)).Append(types.Bool(true)).List(),
		types.String("map"),
		types.NewMap(
			vs,
			types.String("nested"),
			types.String("string")))
	o := NomsValueFromDecodedJSON(vs, map[string]interface{}{
		"string": "string",
		"list":   []interface{}{false, true},
		"map":    map[string]interface{}{"nested": "string"},
	})

	suite.True(m.Equals(o))
}

func (suite *LibTestSuite) TestPanicOnUnsupportedType() {
	vs := suite.vs
	suite.Panics(func() { NomsValueFromDecodedJSON(vs, map[int]string{1: "one"}) }, "Should panic on map[int]string!")
}

func TestFromJSON(t *testing.T) {
	assert := assert.New(t)
	noms := types.NewValueStore((&chunks.TestStorage{}).NewView())

	tests := []struct {
		name    string
		in      string
		want    types.Value
		wantErr string
	}{
		{
			"string",
			`"foo"`,
			types.String("foo"),
			"",
		},
		{
			"ensure canonicalizes",
			`"\u000b"`,
			types.String("\u000B"),
			"",
		},
		{
			"map",
			`{"foo": "bar"}`,
			types.NewMap(noms, types.String("foo"), types.String("bar")),
			"",
		},
		{
			"error: empty value",
			``,
			nil,
			"couldn't parse value '' as json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromJSON([]byte(tt.in), noms)
			if tt.wantErr != "" {
				assert.Error(err)
				assert.Regexp(tt.wantErr, err.Error())
			} else {
				assert.NoError(err, tt.name)
				gotVal := "<nil>"
				if got != nil {
					gotVal = fmt.Sprintf("%s", types.EncodedValue(got))
				}
				assert.True(tt.want.Equals(got), "%s: want %s got %s", tt.name, types.EncodedValue(tt.want), gotVal)
			}
		})
	}
}
