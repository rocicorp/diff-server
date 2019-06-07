package db

import (
	"errors"
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/attic-labs/noms/go/util/json"

	"github.com/aboodman/replicant/util/chk"
)

const (
	LOCAL_DATASET  = "local"
	REMOTE_DATASET = "remote"
)

var (
	schema = nomdl.MustParseType(`
Struct Commit {
	parents: Set<Ref<Cycle<Commit>>>,
	meta: Struct {
		date: Struct DateTime {
			secSinceEpoch: Number,
		},
		op: Struct Tx {
			origin: String,
			code: Ref<Blob>,
			name: String,
			args: List<Value>,
		} |
		Struct Reorder {
			Target: Ref<Cycle<Commit>>,
		} |
		Struct Reject {
			Target: Ref<Cycle<Commit>>,
			Detail: Value
		},
	},
	value: Struct {
		code?: Ref<Blob>,
		data?: Ref<Map<String, Value>>,
	},
}`)

	errCodeNotFound = errors.New("not found")
)

// Not thread-safe
// TODO: need to think carefully about concurrency here
type DB struct {
	db       datas.Database
	prevHead types.Value
	prevData types.Map
	prevCode types.Blob
	data     *types.MapEditor
	code     types.Blob
}

func Load(sp spec.Spec) (*DB, error) {
	if !sp.Path.IsEmpty() {
		return nil, errors.New("Can only load databases from database specs")
	}
	return loadImpl(sp.GetDatabase(), spec.AbsolutePath{
		Dataset: LOCAL_DATASET,
	})
}

func loadImpl(db datas.Database, path spec.AbsolutePath) (*DB, error) {
	headNoms := path.Resolve(db)

	init := func(db *DB) *DB {
		db.data = db.prevData.Edit()
		db.code = db.prevCode
		return db
	}

	if headNoms == nil {
		return init(&DB{
			db:       db,
			prevHead: headNoms,
			prevData: types.NewMap(db),
			prevCode: types.NewEmptyBlob(db),
		}), nil
	}

	headType := types.TypeOf(headNoms)
	if !types.IsSubtype(schema, headType) {
		return &DB{}, fmt.Errorf("Cannot load database. Specified head has non-Replicant data of type: %s", headType.Describe())
	}

	var head Commit
	err := marshal.Unmarshal(headNoms, &head)
	if err != nil {
		return nil, err
	}

	r := &DB{
		db:       db,
		prevHead: headNoms,
	}
	if head.Value.Data.IsZeroValue() {
		r.prevData = types.NewMap(db)
	} else {
		r.prevData = head.Value.Data.TargetValue(db).(types.Map)
	}
	if head.Value.Code.IsZeroValue() {
		r.prevCode = types.NewEmptyBlob(db)
	} else {
		r.prevCode = head.Value.Code.TargetValue(db).(types.Blob)
	}

	return init(r), nil
}

func (db DB) Fork(from hash.Hash) (*DB, error) {
	return loadImpl(db.db, spec.AbsolutePath{
		Hash: from,
	})
}

func (db DB) Head() types.Value {
	return db.prevHead
}

func (db DB) HeadCommit() Commit {
	var c Commit
	marshal.MustUnmarshal(db.Head(), &c)
	return c
}

func (db DB) Noms() datas.Database {
	return db.db
}

func (db *DB) Put(id string, r io.Reader) error {
	v, err := json.FromJSON(r, db.db, json.FromOptions{})
	if err != nil {
		return fmt.Errorf("Invalid JSON: %s", err.Error())
	}
	if v == nil {
		return errors.New("Cannot write null")
	}
	db.data.Set(types.String(id), v)
	return nil
}

func (db *DB) Has(id string) (bool, error) {
	return db.data.Has(types.String(id)), nil
}

func (db *DB) Get(id string, w io.Writer) (bool, error) {
	vv := db.data.Get(types.String(id))
	if vv == nil {
		return false, nil
	}
	v := vv.Value()
	err := json.ToJSON(v, w, json.ToOptions{
		Lists:  true,
		Maps:   true,
		Indent: "",
	})
	if err != nil {
		return false, fmt.Errorf("Key '%s' has non-Replicant data of type: %s", id, types.TypeOf(v).Describe())
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) Del(id string) (bool, error) {
	if !db.data.Has(types.String(id)) {
		return false, nil
	}
	db.data.Remove(types.String(id))
	return true, nil
}

func (db *DB) PutCode(b types.Blob) error {
	db.code = b
	return nil
}

func (db *DB) GetCode() (types.Blob, error) {
	if db.code.Empty() {
		return types.Blob{}, errors.New("no code bundle is registered")
	}
	return db.code, nil
}

func (db *DB) MakeTx(origin string, code types.Blob, fn string, args types.List, date datetime.DateTime) (c Commit, changed bool, err error) {
	newData := db.data.Map()
	newCode := db.code

	if db.prevData.Equals(newData) && db.prevCode.Equals(newCode) {
		return Commit{}, false, nil
	}

	var h Commit
	if db.prevHead != nil {
		h.Parents = append(h.Parents, types.NewRef(db.prevHead))
	}
	h.Meta.Date = date
	h.Meta.Tx.Origin = origin
	h.Meta.Tx.Code = db.db.WriteValue(code)
	h.Meta.Tx.Name = fn
	h.Meta.Tx.Args = args
	h.Value.Data = db.db.WriteValue(newData)
	h.Value.Code = db.db.WriteValue(newCode)
	h.Original = marshal.MustMarshal(db.db, h).(types.Struct)

	return h, true, nil
}

func (db *DB) MakeReorder(target Commit, date datetime.DateTime) (Commit, error) {
	r := Commit{}
	r.Parents = []types.Ref{types.NewRef(target.Original)}
	ontoRef, _ := db.Noms().GetDataset(LOCAL_DATASET).MaybeHeadRef()
	if !ontoRef.IsZeroValue() {
		r.Parents = append(r.Parents, ontoRef)
	}
	r.Meta.Date = date
	r.Meta.Reorder.Subject = types.NewRef(target.Original)
	r.Value.Code = db.Noms().WriteValue(db.code)
	r.Value.Data = db.Noms().WriteValue(db.data.Map())
	r.Original = marshal.MustMarshal(db.db, r).(types.Struct)
	return r, nil
}

func (db *DB) Commit(c Commit) (types.Ref, error) {
	// FastForward not strictly needed here because we should have already ensured that we were
	// fast-forwarding outside of Noms, but it's a nice sanity check.
	noms, err := marshal.Marshal(db.Noms(), c)
	if err != nil {
		return types.Ref{}, err
	}
	r := db.Noms().WriteValue(noms)
	_, err = db.db.FastForward(db.db.GetDataset(LOCAL_DATASET), r)
	if err != nil {
		return types.Ref{}, err
	}
	db.prevHead = noms
	return r, nil
}

type Tx struct {
	Origin string
	Code   types.Ref
	Name   string
	Args   types.List
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
		// TODO: Date should maybe become part of tx, since date of reorder/reject is server-node specific.
		Date datetime.DateTime
		// TODO: Maybe change to "source"? "invoke"? "run"?
		Tx      Tx      `noms:",omitempty"`
		Reorder Reorder `noms:",omitempty"`
		Reject  Reject  `noms:",omitempty"`
	}
	Value struct {
		Data types.Ref `noms:",omitempty"`
		Code types.Ref `noms:",omitempty"`
	}
	Original types.Struct `noms:",original"`
}

type CommitType uint8

const (
	CommitTypeNone = iota
	CommitTypeTx
	CommitTypeReorder
	CommitTypeReject
)

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
	return CommitTypeNone
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

func (c Commit) Basis() types.Ref {
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
	if !found {
		return nil, errors.New("One of meta.{tx, reorder, reject} must be set")
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
		return errors.New("Required field 'op' not present")
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
