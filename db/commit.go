package db

import (
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
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
		Data types.Ref `noms:",omitempty"`
	}
	Original types.Struct `noms:",original"`
}

func (c Commit) Ref() types.Ref {
	return types.NewRef(c.Original)
}

func (c Commit) Data(noms types.ValueReadWriter) types.Map {
	return c.Value.Data.TargetValue(noms).(types.Map)
}

func makeCommit(noms types.ValueReadWriter, basis types.Ref, d datetime.DateTime, newData types.Ref) Commit {
	c := Commit{}
	if !basis.TargetHash().IsEmpty() {
		c.Parents = []types.Ref{basis}
	}
	c.Meta.Date = d
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}
