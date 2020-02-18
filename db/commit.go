package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"roci.dev/replicant/util/chk"
	"roci.dev/replicant/util/noms/union"
)

var (
	schema = nomdl.MustParseType(`
Struct Commit {
	parents: Set<Ref<Cycle<Commit>>>,
	// TODO: It would be cool to call this field "op" or something, but Noms requires a "meta"
	// top-level field.
	meta: Struct Genesis {
		serverCommitID?: String,  // only used on client
	} |
	Struct Tx {
		date:   Struct DateTime {
			secSinceEpoch: Number,
		},
		name: String,
		args: List<Value>,
	} |
	Struct Reorder {
		date:   Struct DateTime {
			secSinceEpoch: Number,
		},
		subject: Ref<Cycle<Commit>>,
	},
	value: Struct {
		data: Ref<Map<String, Value>>,
	},
}`)
)

// TODO: These types should be private
type Tx struct {
	Date datetime.DateTime
	Name string
	Args types.List
}

type Reorder struct {
	Date    datetime.DateTime
	Subject types.Ref
}

type Genesis struct {
	ServerCommitID string `noms:"serverCommitID,omitempty"`
}

type Meta struct {
	// At most one of these will be set. If none are set, then the commit is the genesis commit.
	Tx      Tx      `noms:",omitempty"`
	Reorder Reorder `noms:",omitempty"`
	Genesis Genesis `noms:",omitempty"`
}

func (m Meta) MarshalNoms(vrw types.ValueReadWriter) (val types.Value, err error) {
	v, err := union.Marshal(m, vrw)
	if err != nil {
		return nil, err
	}
	if v == nil {
		v = types.NewStruct("Genesis", types.StructData{})
	}
	return v, nil
}

func (m *Meta) UnmarshalNoms(v types.Value) error {
	return union.Unmarshal(v, m)
}

type Commit struct {
	Parents []types.Ref `noms:",set"`
	Meta    Meta
	Value   struct {
		Data types.Ref `noms:",omitempty"`
	}
	Original types.Struct `noms:",original"`
}

type CommitType uint8

const (
	CommitTypeGenesis = iota
	CommitTypeTx
	CommitTypeReorder
)

func (t CommitType) String() string {
	switch t {
	case CommitTypeGenesis:
		return "CommitTypeGenesis"
	case CommitTypeTx:
		return "CommitTypeTx"
	case CommitTypeReorder:
		return "CommitTypeReorder"
	}
	chk.Fail("NOTREACHED")
	return ""
}

func (c Commit) Ref() types.Ref {
	return types.NewRef(c.Original)
}

func (c Commit) Data(noms types.ValueReadWriter) types.Map {
	return c.Value.Data.TargetValue(noms).(types.Map)
}

func (c Commit) Type() CommitType {
	if c.Meta.Tx.Name != "" {
		return CommitTypeTx
	}
	if !c.Meta.Reorder.Subject.IsZeroValue() {
		return CommitTypeReorder
	}
	return CommitTypeGenesis
}

// TODO: Rename to Subject to avoid confusion with ref.TargetValue().
func (c Commit) Target() types.Ref {
	if !c.Meta.Reorder.Subject.IsZeroValue() {
		return c.Meta.Reorder.Subject
	}
	return types.Ref{}
}

func (c Commit) InitalCommit(noms types.ValueReader) (Commit, error) {
	switch c.Type() {
	case CommitTypeTx, CommitTypeGenesis:
		return c, nil
	case CommitTypeReorder:
		var t Commit
		err := marshal.Unmarshal(c.Target().TargetValue(noms), &t)
		if err != nil {
			return Commit{}, err
		}
		return t.InitalCommit(noms)
	}
	return Commit{}, fmt.Errorf("Unexpected commit of type %v: %s", c.Type(), types.EncodedValue(c.Original))
}

func (c Commit) TargetValue(noms types.ValueReadWriter) types.Value {
	t := c.Target()
	if t.IsZeroValue() {
		return nil
	}
	return t.TargetValue(noms)
}

func (c Commit) TargetCommit(noms types.ValueReadWriter) (Commit, error) {
	tv := c.TargetValue(noms)
	if tv == nil {
		return Commit{}, nil
	}
	var r Commit
	err := marshal.Unmarshal(tv, &r)
	return r, err
}

func (c Commit) BasisRef() types.Ref {
	switch len(c.Parents) {
	case 0:
		return types.Ref{}
	case 1:
		return c.Parents[0]
	case 2:
		subj := c.Target()
		if subj.IsZeroValue() {
			chk.Fail("Unexpected 2-parent type of commit with hash: %s", c.Original.Hash().String())
		}
		for _, p := range c.Parents {
			if !p.Equals(subj) {
				return p
			}
		}
		chk.Fail("Unexpected state for commit with hash: %s", c.Original.Hash().String())
	}
	chk.Fail("Unexpected number of parents (%d) for commit with hash: %s", len(c.Parents), c.Original.Hash().String())
	return types.Ref{}
}

func (c Commit) BasisValue(noms types.ValueReader) types.Value {
	r := c.BasisRef()
	if r.IsZeroValue() {
		return nil
	}
	return r.TargetValue(noms)
}

func (c Commit) Basis(noms types.ValueReader) (Commit, error) {
	var r Commit
	err := marshal.Unmarshal(c.BasisValue(noms), &r)
	if err != nil {
		return Commit{}, err
	}
	return r, nil
}

func makeGenesis(noms types.ValueReadWriter, serverCommitID string) Commit {
	c := Commit{}
	c.Meta.Genesis.ServerCommitID = serverCommitID
	c.Value.Data = noms.WriteValue(types.NewMap(noms))
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	noms.WriteValue(c.Original)
	return c
}

func makeTx(noms types.ValueReadWriter, basis types.Ref, d datetime.DateTime, f string, args types.List, newData types.Ref) Commit {
	c := Commit{}
	c.Parents = []types.Ref{basis}
	c.Meta.Tx.Date = d
	c.Meta.Tx.Name = f
	c.Meta.Tx.Args = args
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}

func makeReorder(noms types.ValueReadWriter, basis types.Ref, d datetime.DateTime, subject, newData types.Ref) Commit {
	c := Commit{}
	c.Parents = []types.Ref{basis, subject}
	c.Meta.Reorder.Date = d
	c.Meta.Reorder.Subject = subject
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}
