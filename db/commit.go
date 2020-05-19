package db

import (
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"roci.dev/diff-server/kv"
)

var (
	schema = nomdl.MustParseType(`
Struct Commit {
	parents: Set<Ref<Cycle<Commit>>>,
	meta: Struct {
		date:   Struct DateTime {
			secSinceEpoch: Number,
		},
	},
	value: Struct {
		checksum: String,
		lastMutationID: Number,
		data: Ref<Map<String, Value>>,
	},
}`)
)

type Commit struct {
	Parents []types.Ref `noms:",set"`
	Meta    struct {
		Date datetime.DateTime
	}
	Value struct {
		Checksum       types.String
		LastMutationID types.Number
		Data           types.Ref `noms:",omitempty"`
	}
	Original types.Struct `noms:",original"`
}

func (c Commit) Ref() types.Ref {
	return types.NewRef(c.Original)
}

func (c Commit) Data(noms types.ValueReadWriter) kv.Map {
	return kv.FromNoms(noms, c.Value.Data.TargetValue(noms).(types.Map), kv.MustChecksumFromString(string(c.Value.Checksum)))
}

// Basis returns the basis (parent) of the Commit.
func (c Commit) Basis(noms types.ValueReadWriter) (Commit, error) {
	return Read(noms, c.Parents[0].TargetHash())
}

func makeCommit(noms types.ValueReadWriter, basis types.Ref, d datetime.DateTime, newData types.Ref, checksum types.String, lastMutationID uint64) Commit {
	c := Commit{}
	if !basis.IsZeroValue() {
		c.Parents = []types.Ref{basis}
	}
	c.Meta.Date = d
	c.Value.Checksum = checksum
	c.Value.LastMutationID = types.Number(lastMutationID) // Warning: potentially lossy!
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}
