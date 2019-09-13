package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/util/noms/diff"
)

func TestMarshal(t *testing.T) {
	assert := assert.New(t)

	noms := types.NewValueStore((&chunks.TestStorage{}).NewView())
	emptyBlob := noms.WriteValue(types.NewEmptyBlob(noms))
	emptyMap := noms.WriteValue(types.NewMap(noms))

	d := datetime.Now()
	br := noms.WriteValue(types.NewBlob(noms, strings.NewReader("blobdata")))
	dr := noms.WriteValue(types.NewMap(noms, types.String("foo"), types.String("bar")))
	args := types.NewList(noms, types.Bool(true), types.String("monkey"))
	g := makeGenesis(noms)
	tx := makeTx(noms, types.NewRef(g.Original), "o1", d, emptyBlob, "func", args, br, dr)
	noms.WriteValue(g.Original)
	noms.WriteValue(tx.Original)

	tc := []struct {
		in  Commit
		exp types.Value
	}{
		{
			makeGenesis(noms),
			types.NewStruct("Commit", types.StructData{
				"meta":    types.NewStruct("Genesis", types.StructData{}),
				"parents": types.NewSet(noms),
				"value": types.NewStruct("", types.StructData{
					"code": emptyBlob,
					"data": emptyMap,
				}),
			}),
		},
		{
			tx,
			types.NewStruct("Commit", types.StructData{
				"parents": types.NewSet(noms, types.NewRef(g.Original)),
				"meta": types.NewStruct("Tx", types.StructData{
					"origin": types.String("o1"),
					"date":   marshal.MustMarshal(noms, d),
					"code":   emptyBlob,
					"name":   types.String("func"),
					"args":   args,
				}),
				"value": types.NewStruct("", types.StructData{
					"code": br,
					"data": dr,
				}),
			}),
		},
		{
			makeTx(noms, types.NewRef(g.Original), "o1", d, br, "func", args, emptyBlob, dr),
			types.NewStruct("Commit", types.StructData{
				"parents": types.NewSet(noms, types.NewRef(g.Original)),
				"meta": types.NewStruct("Tx", types.StructData{
					"origin": types.String("o1"),
					"date":   marshal.MustMarshal(noms, d),
					"code":   br,
					"name":   types.String("func"),
					"args":   args,
				}),
				"value": types.NewStruct("", types.StructData{
					"code": emptyBlob,
					"data": dr,
				}),
			}),
		},
		{
			makeReorder(noms, types.NewRef(g.Original), "o1", d, types.NewRef(tx.Original), br, dr),
			types.NewStruct("Commit", types.StructData{
				"parents": types.NewSet(noms, types.NewRef(g.Original), types.NewRef(tx.Original)),
				"meta": types.NewStruct("Reorder", types.StructData{
					"origin":  types.String("o1"),
					"date":    marshal.MustMarshal(noms, d),
					"subject": types.NewRef(tx.Original),
				}),
				"value": types.NewStruct("", types.StructData{
					"code": br,
					"data": dr,
				}),
			}),
		},
		{
			makeReject(noms, types.NewRef(g.Original), "o1", d, types.NewRef(tx.Original), "didn't feel like it", br, dr),
			types.NewStruct("Commit", types.StructData{
				"parents": types.NewSet(noms, types.NewRef(g.Original), types.NewRef(tx.Original)),
				"meta": types.NewStruct("Reject", types.StructData{
					"origin":  types.String("o1"),
					"date":    marshal.MustMarshal(noms, d),
					"subject": types.NewRef(tx.Original),
					"reason":  types.String("didn't feel like it"),
				}),
				"value": types.NewStruct("", types.StructData{
					"code": br,
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
