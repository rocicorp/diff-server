package db

import (
	"errors"
	"fmt"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/util/chk"
)

// TODO: These types should be private
type Tx struct {
	Code   types.Ref `noms:",omitempty"`  // TODO: rename: "Bundle/BundleRef"
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
	Subject types.Ref
}

type Reject struct {
	Subject types.Ref
	Reason  string
}

type Commit struct {
	Parents []types.Ref `noms:",set"`
	Meta    struct {
		Origin string
		// TODO: Date should maybe become part of tx, since date of reorder/reject is server-node specific.
		Date datetime.DateTime
		// TODO: Maybe change to "source"? "invoke"? "run"?
		Tx      Tx      `noms:",omitempty"`
		Reorder Reorder `noms:",omitempty"`
		Reject  Reject  `noms:",omitempty"`
	}
	Value struct {
		Code types.Ref `noms:",omitempty"`
		Data types.Ref `noms:",omitempty"`  // TODO: Rename "Bundle"
	}
	Original types.Struct `noms:",original"`

	data   types.Map  `noms:"-"`
	bundle types.Blob `noms:"-"`
}

type CommitType uint8

const (
	CommitTypeGenesis = iota
	CommitTypeTx
	CommitTypeReorder
	CommitTypeReject
)

func (c Commit) Data(noms types.ValueReadWriter) types.Map {
	if c.data == (types.Map{}) {
		c.data = c.Value.Data.TargetValue(noms).(types.Map)
	}
	return c.data
}

func (c Commit) Bundle(noms types.ValueReadWriter) types.Blob {
	if c.bundle == (types.Blob{}) {
		c.bundle = c.Value.Code.TargetValue(noms).(types.Blob)
	}
	return c.bundle
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

func (c Commit) FinalReorderTarget(noms types.ValueReader) (Commit, error) {
	switch c.Type() {
	case CommitTypeTx:
		return c, nil
	case CommitTypeReorder:
		return c.FinalReorderTarget(noms)
	default:
		return Commit{}, fmt.Errorf("Unexpected reorder target of type %s: %s", c.Type(), types.EncodedValue(c.Original))
	}
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

func (c Commit) MarshalNoms(vrw types.ValueReadWriter) (val types.Value, err error) {
	r, err := marshal.Marshal(vrw, internal(c))
	if err != nil {
		return nil, err
	}
	rs := r.(types.Struct)
	meta := rs.Get("meta").(types.Struct)
	var found = false
	for _, f := range []string{"tx", "reorder", "reject"} {
		if v, ok := meta.MaybeGet(f); ok {
			if found {
				return nil, errors.New("Only one of meta.{tx, reorder, reject} may be set")
			}
			meta = meta.Set("op", v.(types.Struct)).Delete(f)
			found = true
		}
	}
	return rs.Set("meta", meta), nil
}

func (c *Commit) UnmarshalNoms(v types.Value) error {
	err := marshal.Unmarshal(v, (*internal)(c))
	if err != nil {
		return err
	}
	op, ok := c.Original.Get("meta").(types.Struct).MaybeGet("op")
	if !ok {
		return nil
	}
	ops, ok := op.(types.Struct)
	if !ok {
		return errors.New("Field 'op' must be a struct")
	}
	switch ops.Name() {
	case "Tx":
		return marshal.Unmarshal(op, &c.Meta.Tx)
	case "Reorder":
		return marshal.Unmarshal(op, &c.Meta.Reorder)
	case "Reject":
		return marshal.Unmarshal(op, &c.Meta.Reject)
	default:
		return fmt.Errorf("Invalid op type: %s", ops.Name())
	}
	return nil
}

type internal Commit

func (_ internal) MarshalNomsStructName() string {
	return "Commit"
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
	c.Meta.Origin = origin
	c.Meta.Date = d
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
	c.Meta.Origin = origin
	c.Meta.Date = d
	c.Meta.Reorder.Subject = subject
	c.Value.Code = newBundle
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}

func makeReject(noms types.ValueReadWriter, basis types.Ref, origin string, d datetime.DateTime, subject types.Ref, reason string, newBundle, newData types.Ref) Commit {
	c := Commit{}
	c.Parents = []types.Ref{basis, subject}
	c.Meta.Origin = origin
	c.Meta.Date = d
	c.Meta.Reject.Subject = subject
	c.Meta.Reject.Reason = reason
	c.Value.Code = newBundle
	c.Value.Data = newData
	c.Original = marshal.MustMarshal(noms, c).(types.Struct)
	return c
}
