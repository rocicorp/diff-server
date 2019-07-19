package db

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/attic-labs/noms/go/diff"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/exec"
	"github.com/aboodman/replicant/util/chk"
)

func TestReplay(t *testing.T) {
	assert := assert.New(t)

	// cases
	// - transaction that doesn't run
	// - transaction that is already at correct (empty) basis
	// - transaction with wrong non-empty basis
	// - transaction with wront empty basis
	// - reordered commit with right basis
	// - reordered commit with wrong basis

	now := time.Now()
	var d *DB
	var fork, serverHead, clientHead Commit
	var expected Commit

	// - transaction already ordered correctly
	d = initDB(assert)
	serverHead = run(d, "a", nil, true)
	clientHead = run(d, "b", &serverHead, false)
	expected = simulate(d, clientHead.Meta.Date, "b", serverHead.Original, []string{"a", "b"})
	merge(d, serverHead.Original.Hash(), now, clientHead, expected, assert)

	d = initDB(assert)
	serverHead = run(d, "a", nil, true)
	serverHead = run(d, "b", &serverHead, true)
	c1 := run(d, "c", &serverHead, false)
	clientHead = run(d, "d", &c1, false)
	expected = simulate(d, c1.Meta.Date, "c", serverHead.Original, []string{"a", "b", "c"})
	expected = simulate(d, clientHead.Meta.Date, "d", c1.Original, []string{"a", "b", "c", "d"})
	merge(d, serverHead.Original.Hash(), now, clientHead, expected, assert)

	// - transaction gets reordered
	d = initDB(assert)
	fork = run(d, "a", nil, true)
	serverHead = run(d, "b", &fork, true)
	clientHead = run(d, "c", &fork, false)
	expected = simulateReorder(d, datetime.DateTime{now}, &serverHead, clientHead, []string{"a", "b", "c"})
	merge(d, serverHead.Original.Hash(), now, clientHead, expected, assert)

	// two-commit branch gets reordered
	d = initDB(assert)
	fork = run(d, "a", nil, true)
	serverHead = run(d, "b", &fork, true)
	clientParent := run(d, "c", &fork, false)
	clientHead = run(d, "d", &clientParent, false)
	simulatedClientParent := simulateReorder(d, datetime.DateTime{now}, &serverHead, clientParent, []string{"a", "b", "c"})
	simulatedClient := simulateReorder(d, datetime.DateTime{now}, &simulatedClientParent, clientHead, []string{"a", "b", "c", "d"})
	merge(d, serverHead.Original.Hash(), now, clientHead, simulatedClient, assert)
}

func merge(d *DB, forkPoint hash.Hash, now time.Time, clientHead, expected Commit, assert *assert.Assertions) {
	d, err := d.Fork(forkPoint)
	assert.NoError(err)
	ld := LocalDest{db: d}
	ld.now = func() time.Time {
		return now
	}
	_, err = ld.Merge(clientHead.Original.Hash())
	assert.NoError(err)
	if !d.Head().Equals(expected.Original) {
		buf := &bytes.Buffer{}
		err = diff.PrintDiff(buf, expected.Original, d.Head(), false)
		assert.NoError(err)
		assert.Fail("Diff: %s", buf.String())
	}
}

func initDB(assert *assert.Assertions) *DB {
	d, dir := LoadTempDB(assert)
	fmt.Println("testdb: ", dir)

	cmd := CodePut{}
	cmd.In.Origin = "c1"
	cmd.InStream = types.NewBlob(d.Noms(), strings.NewReader(`
function append(id, item) {
	var old = db.get(id) || [];
	old.push(item);
	db.put(id, old);
}
`)).Reader()
	err := cmd.Run(d)
	chk.NoError(err)

	return d
}

func run(d *DB, item string, basis *Commit, doCommit bool) Commit {
	var err error
	if basis != nil {
		d, err = d.Fork(basis.Original.Hash())
		chk.NoError(err)
	}
	code, err := d.GetCode()
	chk.NoError(err)
	args := types.NewList(d.Noms(), types.String("o1"), types.String(item))
	err = exec.Run(d, code.Reader(), "append", args)
	chk.NoError(err)
	commit, changes, err := d.MakeTx("c1", code, "append", args, datetime.Now())
	chk.True(changes, "")
	chk.NoError(err)
	_ = d.Noms().WriteValue(commit.Original)
	if doCommit {
		_, err = d.Commit(commit)
		chk.NoError(err)
	}
	return commit
}

func simulate(d *DB, date datetime.DateTime, item string, basis types.Value, value []string) Commit {
	code, err := d.GetCode()
	chk.NoError(err)
	args := types.NewList(d.Noms(), types.String("o1"), types.String(item))
	r := Commit{}
	if basis != nil {
		r.Parents = []types.Ref{types.NewRef(basis)}
	}
	r.Meta.Date = date
	r.Meta.Tx.Origin = "c1"
	r.Meta.Tx.Code = types.NewRef(code)
	r.Meta.Tx.Name = "append"
	r.Meta.Tx.Args = args
	r.Value.Code = types.NewRef(code)
	r.Value.Data = d.Noms().WriteValue(types.NewMap(d.Noms(), types.String("o1"), val(d.Noms(), value)))
	r.Original = marshal.MustMarshal(d.Noms(), r).(types.Struct)
	return r
}

func val(noms types.ValueReadWriter, s []string) types.List {
	r := types.NewList(noms).Edit()
	for _, ss := range s {
		r.Append(types.String(ss))
	}
	return r.List()
}

func simulateReorder(d *DB, date datetime.DateTime, basis *Commit, target Commit, value []string) Commit {
	d, err := d.Fork(basis.Original.Hash())
	chk.NoError(err)
	r := Commit{}
	r.Parents = append(r.Parents, types.NewRef(target.Original))
	if basis != nil {
		r.Parents = append(r.Parents, types.NewRef(basis.Original))
	}
	r.Meta.Date = date
	r.Meta.Reorder.Subject = types.NewRef(target.Original)
	r.Value.Code = basis.Value.Code
	r.Value.Data = d.Noms().WriteValue(types.NewMap(d.Noms(), types.String("o1"), val(d.Noms(), value)))
	r.Original = marshal.MustMarshal(d.Noms(), r).(types.Struct)
	return r
}

/*
func TestLocalDest(t *testing.T) {
	assert := assert.New(t)

	d, dir := db.LoadTempDB(assert)
	fmt.Println("testdb: ", dir)

	// cases:
	// - fast-forward client
	// - fast-forward server
	// - null sync
	// - no client head
	// - no server head
	// - true merge
	// - client includes reorder
}
*/
