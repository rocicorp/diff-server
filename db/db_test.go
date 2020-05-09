package db

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/kv"
)

func TestGenesis(t *testing.T) {
	assert := assert.New(t)
	db, _ := LoadTempDB(assert)
	assert.False(db.Hash().IsEmpty())
	assert.True(db.Head().Data(db.Noms()).Empty())
}

func TestPutData(t *testing.T) {
	assert := assert.New(t)
	db, _ := LoadTempDB(assert)
	genesis := db.Head()
	me := kv.NewMap(db.Noms()).Edit()
	assert.NoError(me.Set("key", types.Bool(true)))
	m := me.Build()

	c1, err := db.PutData(m, 1)
	assert.NoError(err)
	assert.False(genesis.Original.Equals(c1.Original))
	assert.True(c1.Original.Equals(db.Head().Original))
	assert.True(m.NomsMap().Value().Equals(c1.Data(db.Noms())))
	assert.True(types.Number(1).Equals(c1.Value.LastMutationID))

	c2, err := db.PutData(m, 2)
	assert.NoError(err)
	assert.True(c2.Original.Equals(db.Head().Original))
	assert.True(types.Number(2).Equals(c2.Value.LastMutationID))
}

// hmmm.. we seem to have removed most tests.
