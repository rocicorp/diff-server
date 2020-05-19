package db

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/noms/diff"
)

func TestBasis(t *testing.T) {
	assert := assert.New(t)
	db, _ := LoadTempDB(assert)
	genesis := db.Head()
	c, err := db.MaybePutData(kv.NewMap(db.Noms()), 2)
	assert.NoError(err)
	if err == nil {
		assert.False(c.NomsStruct.IsZeroValue())
	}
	basis, err := c.Basis(db.Noms())
	assert.NoError(err)
	assert.True(genesis.NomsStruct.Equals(basis.NomsStruct))
}

func TestMarshal(t *testing.T) {
	assert := assert.New(t)

	noms := types.NewValueStore((&chunks.TestStorage{}).NewView())
	emptyMap := noms.WriteValue(types.NewMap(noms))
	checksum1 := types.String("1")
	lastMutationID1 := uint64(1)

	d := datetime.Now()
	dr := noms.WriteValue(types.NewMap(noms, types.String("foo"), types.String("bar")))
	checksum2 := types.String("2")
	lastMutationID2 := uint64(2)
	c1 := makeCommit(noms, types.Ref{}, d, noms.WriteValue(types.NewMap(noms)), checksum1, lastMutationID1)
	c2 := makeCommit(noms, noms.WriteValue(c1.NomsStruct), d, dr, checksum2, lastMutationID2)
	noms.WriteValue(c2.NomsStruct)

	tc := []struct {
		in  Commit
		exp types.Value
	}{
		{
			c1,
			types.NewStruct("Commit", types.StructData{
				"meta": types.NewStruct("", types.StructData{
					"date": marshal.MustMarshal(noms, d),
				}),
				"parents": types.NewSet(noms),
				"value": types.NewStruct("", types.StructData{
					"checksum":       types.String("1"),
					"data":           emptyMap,
					"lastMutationID": types.Number(lastMutationID1),
				}),
			}),
		},
		{
			c2,
			types.NewStruct("Commit", types.StructData{
				"parents": types.NewSet(noms, c1.Ref()),
				"meta": types.NewStruct("", types.StructData{
					"date": marshal.MustMarshal(noms, d),
				}),
				"value": types.NewStruct("", types.StructData{
					"checksum":       types.String("2"),
					"data":           dr,
					"lastMutationID": types.Number(lastMutationID2),
				}),
			}),
		},
	}

	for i, t := range tc {
		act, err := marshal.Marshal(noms, t.in)
		assert.NoError(err, "test case: %d", i)
		assert.True(t.exp.Equals(act), "test case: %d - %s", i, diff.Diff(t.exp, act))

		var roundtrip Commit
		err = marshal.Unmarshal(act, &roundtrip)
		assert.NoError(err)

		remarshalled, err := marshal.Marshal(noms, roundtrip)
		assert.NoError(err)
		assert.True(act.Equals(remarshalled), fmt.Sprintf("test case %d", i))
	}
}
