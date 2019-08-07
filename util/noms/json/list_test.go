package json

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/util/noms/memstore"
)

func TestListUnmarshal(t *testing.T) {
	assert := assert.New(t)
	vs := memstore.New()

	var l List
	l.Noms = vs

	err := l.UnmarshalJSON([]byte("42"))
	assert.EqualError(err, "Unexpected Noms type: Number")

	assert.Nil(l.Value.Value)
	assert.Panics(func() {
		l.List()
	})

	err = l.UnmarshalJSON([]byte("[42]"))
	assert.NoError(err)
	expected := types.NewList(vs, types.Number(42))
	assert.True(expected.Equals(l.Value.Value))
	assert.True(expected.Equals(l.List()))
}
