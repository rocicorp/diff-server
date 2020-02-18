package db

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"

	"roci.dev/replicant/util/noms/diff"
)

func TestMarshal(t *testing.T) {
	assert := assert.New(t)

	noms := types.NewValueStore((&chunks.TestStorage{}).NewView())
	emptyMap := noms.WriteValue(types.NewMap(noms))

	d := datetime.Now()
	dr := noms.WriteValue(types.NewMap(noms, types.String("foo"), types.String("bar")))
	args := types.NewList(noms, types.Bool(true), types.String("monkey"))
	g := makeGenesis(noms, "")
	tx := makeTx(noms, types.NewRef(g.Original), d, "func", args, dr)
	noms.WriteValue(g.Original)
	noms.WriteValue(tx.Original)

	tc := []struct {
		in  Commit
		exp types.Value
	}{
		{
			makeGenesis(noms, ""),
			types.NewStruct("Commit", types.StructData{
				"meta":    types.NewStruct("Genesis", types.StructData{}),
				"parents": types.NewSet(noms),
				"value": types.NewStruct("", types.StructData{
					"data": emptyMap,
				}),
			}),
		},
		{
			makeGenesis(noms, "foo"),
			types.NewStruct("Commit", types.StructData{
				"meta": types.NewStruct("Genesis", types.StructData{
					"serverCommitID": types.String("foo"),
				}),
				"parents": types.NewSet(noms),
				"value": types.NewStruct("", types.StructData{
					"data": emptyMap,
				}),
			}),
		},
		{
			tx,
			types.NewStruct("Commit", types.StructData{
				"parents": types.NewSet(noms, types.NewRef(g.Original)),
				"meta": types.NewStruct("Tx", types.StructData{
					"date": marshal.MustMarshal(noms, d),
					"name": types.String("func"),
					"args": args,
				}),
				"value": types.NewStruct("", types.StructData{
					"data": dr,
				}),
			}),
		},
		{
			makeTx(noms, types.NewRef(g.Original), d, "func", args, dr),
			types.NewStruct("Commit", types.StructData{
				"parents": types.NewSet(noms, types.NewRef(g.Original)),
				"meta": types.NewStruct("Tx", types.StructData{
					"date": marshal.MustMarshal(noms, d),
					"name": types.String("func"),
					"args": args,
				}),
				"value": types.NewStruct("", types.StructData{
					"data": dr,
				}),
			}),
		},
		{
			makeReorder(noms, types.NewRef(g.Original), d, types.NewRef(tx.Original), dr),
			types.NewStruct("Commit", types.StructData{
				"parents": types.NewSet(noms, types.NewRef(g.Original), types.NewRef(tx.Original)),
				"meta": types.NewStruct("Reorder", types.StructData{
					"date":    marshal.MustMarshal(noms, d),
					"subject": types.NewRef(tx.Original),
				}),
				"value": types.NewStruct("", types.StructData{
					"data": dr,
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
