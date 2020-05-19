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

func TestRead(t *testing.T) {
	assert := assert.New(t)
	db, _ := LoadTempDB(assert)
	c, err := Read(db.Noms(), db.Hash())
	assert.NoError(err)
	assert.True(db.Head().NomsStruct.Equals(c.NomsStruct))
}

func TestMaybePutData(t *testing.T) {
	assert := assert.New(t)
	db, _ := LoadTempDB(assert)
	genesis := db.Head()
	me := kv.NewMap(db.Noms()).Edit()
	assert.NoError(me.Set("key", types.Bool(true)))
	m := me.Build()

	c1, err := db.MaybePutData(m, 1)
	assert.NoError(err)
	assert.False(genesis.NomsStruct.Equals(c1.NomsStruct))
	assert.True(c1.NomsStruct.Equals(db.Head().NomsStruct))
	assert.True(m.NomsMap().Value().Equals(c1.Data(db.Noms())))
	assert.True(types.Number(1).Equals(c1.Value.LastMutationID))

	c2, err := db.MaybePutData(m, 2)
	assert.NoError(err)
	assert.True(c2.NomsStruct.Equals(db.Head().NomsStruct))
	assert.True(types.Number(2).Equals(c2.Value.LastMutationID))

	c3, err := db.MaybePutData(m, 2)
	assert.NoError(err)
	assert.True(c3.NomsStruct.IsZeroValue())
	assert.True(c2.NomsStruct.Equals(db.Head().NomsStruct))
}

// hmmm.. we seem to have removed most tests.
