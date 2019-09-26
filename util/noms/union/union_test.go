package union

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/util/noms/memstore"
)

func TestMarshalUnion(t *testing.T) {
	assert := assert.New(t)

	type Bar struct {
		A string
	}

	type Baz struct {
		B string
	}

	type Foo struct {
		Bar Bar
		Baz Baz
	}

	f := Foo{}
	f.Baz.B = "monkey"

	ms := memstore.New()

	exp := types.NewStruct("Baz", types.StructData{"b": types.String("monkey")})

	v, err := Marshal(f, ms)
	assert.NoError(err)
	assert.True(exp.Equals(v))

	act := Foo{}
	err = Unmarshal(v, &act)
	assert.NoError(err)
	assert.Equal(act, f)

	f.Bar.A = "a"
	v, err = Marshal(f, ms)
	assert.EqualError(err, "At most one field of a union may be set")
	assert.Nil(v)
}
