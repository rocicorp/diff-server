package db

import (
	"errors"
	"fmt"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/util/chk"
	"github.com/aboodman/replicant/util/noms/union"
)

var (
	schema = nomdl.MustParseType(`
Struct Commit {
	parents: Set<Ref<Cycle<Commit>>>,
	// TODO: It would be cool to call this field "op" or something, but Noms requires a "meta"
	// top-level field.
	meta: Struct Genesis {
	} |
	Struct Tx {
		origin: String,
		date:   Struct DateTime {
			secSinceEpoch: Number,
		},
		code?: Ref<Blob>,	// omitted for system functions
		name: String,
		args: List<Value>,
	} |
	Struct Reorder {
		origin: String,
		date:   Struct DateTime {
			secSinceEpoch: Number,
		},
		subject: Ref<Cycle<Commit>>,
	} |
	Struct Reject {
		origin: String,
		date:   Struct DateTime {
			secSinceEpoch: Number,
		},
		subject: Ref<Cycle<Commit>>,
		reason2: Struct Nondeterm {
			expected: Ref<Cycle<Commit>>,
		} | Struct Fiat {
			detail: String,
		},

		reason: Value, // deprecated
	},
	value: Struct {
		code: Ref<Blob>,
		data: Ref<Map<String, Value>>,
	},
}`)
)

// TODO: These types should be private
type Tx struct {
	Origin string
	Date   datetime.DateTime
	Code   types.Ref `noms:",omitempty"` // TODO: rename: "Bundle/BundleRef"
	Name   string
	Args   types.List
}

func (tx Tx) Bundle(noms types.ValueReader) types.Blob {
	if tx.Code.IsZeroValue() {
		return types.Blob{}
	}
	return tx.Code.TargetValue(noms).(types.Blob)
}

type Reorder struct {
	Origin  string
	Date    datetime.DateTime
	Subject types.Ref
}

type Nondeterm struct {
	Expected types.Ref
}

type Fiat struct {
	Detail string
}

type Reason struct {
	Nondeterm Nondeterm
	Fiat      Fiat
}

func (r Reason) MarshalNoms(vrw types.ValueReadWriter) (val types.Value, err error) {
	v, err := union.Marshal(r, vrw)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, errors.New("At least one field of Reason is required")
	}
	return v, nil
}

func (r *Reason) UnmarshalNoms(v types.Value) error {
	return union.Unmarshal(v, r)
}

type Reject struct {
	Origin  string
	Date    datetime.DateTime
	Subject types.Ref
	Reason  Reason `noms:"reason2"`
}

func (r Reject) MarshalNoms(vrw types.ValueReadWriter) (val types.Value, err error) {
	type internal Reject
	vs := marshal.MustMarshal(vrw, internal(r)).(types.Struct)

	// Must set legacy "reason" even though it isn't used for backward compat with clients that validated that.
	vs = vs.Set("reason", types.String("unused"))
	vs = vs.SetName("Reject")

	return vs, nil
}

type Meta struct {
	// At most one of these will be set. If none are set, then the commit is the genesis commit.
	Tx      Tx      `noms:",omitempty"`
	Reorder Reorder `noms:",omitempty"`
	Reject  Reject  `noms:",omitempty"`
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
	if v.(types.Struct).Name() == "Genesis" {
		return nil
	}
	return union.Unmarshal(v, m)
}

type Commit struct {
	Parents []types.Ref `noms:",set"`
	Meta    Meta
	Value   struct {
		Code types.Ref `noms:",omitempty"`
		Data types.Ref `noms:",omitempty"` // TODO: Rename "Bundle"
	}
	Original types.Struct `noms:",original"`
}

type CommitType uint8

const (
	CommitTypeGenesis = iota
	CommitTypeTx
	CommitTypeReorder
	CommitTypeReject
)

func (t CommitType) String() string {
	switch t {
	case CommitTypeGenesis:
		return "CommitTypeGenesis"
	case CommitTypeTx:
		return "CommitTypeTx"
	case CommitTypeReorder:
		return "CommitTypeReorder"
	case CommitTypeReject:
		return "CommitTypeReject"
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

func (c Commit) Bundle(noms types.ValueReadWriter) types.Blob {
	return c.Value.Code.TargetValue(noms).(types.Blob)
}

func (c Commit) Type() CommitType {
	if c.Meta.Tx.Name != "" {
		return CommitTypeTx
	}
	if !c.Meta.Reorder.Subject.IsZeroValue() {
		return CommitTypeReorder
	}
	if !c.Meta.Reject.Subject.IsZeroValue() {
		return CommitTypeReject
	}
	return CommitTypeGenesis
}

// TODO: Rename to Subject to avoid confusion with ref.TargetValue().
func (c Commit) Target() types.Ref {
	if !c.Meta.Reorder.Subject.IsZeroValue() {
		return c.Meta.Reorder.Subject
	} else if !c.Meta.Reject.Subject.IsZeroValue() {
		return c.Meta.Reject.Subject
	}
	return types.Ref{}
}

func (c Commit) InitalCommit(noms types.ValueReader) (Commit, error) {
	switch c.Type() {
	case CommitTypeTx, CommitTypeGenesis:
		return c, nil
	case CommitTypeReorder, CommitTypeReject:
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

func makeGenesis(noms types.ValueReadWriter) Commit {
	c := Commit{}
	c.Value.Data = noms.WriteValue(types.NewMap(noms))
	c.Value.Code = noms.WriteValue(types.NewBlob(noms))
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	noms.WriteValue(c.Original)
	return c
}

func makeTx(noms types.ValueReadWriter, basis types.Ref, origin string, d datetime.DateTime, bundle types.Ref, f string, args types.List, newBundle, newData types.Ref) Commit {
	c := Commit{}
	c.Parents = []types.Ref{basis}
	c.Meta.Tx.Origin = origin
	c.Meta.Tx.Date = d
	c.Meta.Tx.Code = bundle
	c.Meta.Tx.Name = f
	c.Meta.Tx.Args = args
	c.Value.Code = newBundle
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}

func makeReorder(noms types.ValueReadWriter, basis types.Ref, origin string, d datetime.DateTime, subject, newBundle, newData types.Ref) Commit {
	c := Commit{}
	c.Parents = []types.Ref{basis, subject}
	c.Meta.Reorder.Origin = origin
	c.Meta.Reorder.Date = d
	c.Meta.Reorder.Subject = subject
	c.Value.Code = newBundle
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}

func makeReject(noms types.ValueReadWriter, basis types.Ref, origin string, d datetime.DateTime, subject, nondeterm types.Ref, fiatDetail string, newBundle, newData types.Ref) Commit {
	c := Commit{}
	c.Parents = []types.Ref{basis, subject}
	c.Meta.Reject.Origin = origin
	c.Meta.Reject.Date = d
	c.Meta.Reject.Subject = subject
	c.Meta.Reject.Reason.Nondeterm.Expected = nondeterm
	c.Meta.Reject.Reason.Fiat.Detail = fiatDetail
	c.Value.Code = newBundle
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}
