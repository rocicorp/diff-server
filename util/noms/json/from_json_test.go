// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package json

import (
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
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
	suite.False(NomsValueFromDecodedJSON(vs, 1.7).Equals(types.Bool(true)))
}

func (suite *LibTestSuite) TestCompositeTypes() {
	vs := suite.vs

	// [false true]
	suite.EqualValues(
		types.NewList(vs).Edit().Append(types.Bool(false)).Append(types.Bool(true)).List(),
		NomsValueFromDecodedJSON(vs, []interface{}{false, true}))

	// [[false true]]
	suite.EqualValues(
		types.NewList(vs).Edit().Append(
			types.NewList(vs).Edit().Append(types.Bool(false)).Append(types.Bool(true)).List()).List(),
		NomsValueFromDecodedJSON(vs, []interface{}{[]interface{}{false, true}}))

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
