package json

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func TestValueJSONMarshal(t *testing.T) {
	assert := assert.New(t)
	noms := types.NewValueStore((&chunks.TestStorage{}).NewView())

	tc := []struct {
		n types.Value
		j string
	}{
		{types.Bool(true), "true"},
		{types.Bool(false), "false"},
		{types.Number(42), "42"},
		{types.String("foo"), "\"foo\""},
		{types.NewList(noms, types.Bool(true)), "[true]"},
		{types.NewMap(noms, types.String("foo"), types.Bool(true)), "{\"foo\":true}"},
		{Null(), "null"},
	}

	for i, t := range tc {
		msg := fmt.Sprintf("test case %d", i)
		v := New(noms, t.n)
		marshaled, err := v.MarshalJSON()
		assert.NoError(err, msg)
		assert.Equal(t.j, string(marshaled), msg)
		v = New(noms, nil)
		err = v.UnmarshalJSON(marshaled)
		assert.NoError(err, msg)
		assert.True(v.Value.Equals(t.n), msg)
	}
}
