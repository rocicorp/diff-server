package db

import (
	"os"
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/time"
)

func TestReload(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()
	genesis := db.Head()

	// Change head behind db's back.
	db2 := LoadTempDBWithPath(assert, dir)
	me := kv.NewMap(db2.Noms()).Edit()
	assert.NoError(me.Set("key", types.Bool(true)))
	m := me.Build()
	valueRef := db2.Noms().WriteValue(m.NomsMap())
	newCommit := makeCommit(db2.Noms(), types.NewRef(genesis.NomsStruct), time.DateTime(), valueRef, m.NomsChecksum(), 123)
	db2.Noms().WriteValue(newCommit.NomsStruct)
	err := db2.setHead(newCommit)
	assert.NoError(err)
	assert.False(genesis.NomsStruct.Equals(newCommit.NomsStruct))
	assert.True(newCommit.NomsStruct.Equals(db2.Head().NomsStruct))

	// Now check that db picks up the change.
	assert.NoError(db.Reload())
	assert.True(newCommit.NomsStruct.Equals(db.Head().NomsStruct))
}

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
