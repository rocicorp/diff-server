package db

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/log"
)

func TestDiff(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)
	fmt.Println(dir)

	var fromID hash.Hash
	var fromChecksum string
	tc := []struct {
		label         string
		f             func()
		expectedDiff  []kv.Operation
		expectedError string
	}{
		{
			"same-commit",
			func() {},
			[]kv.Operation{},
			"",
		},
		{
			"change-1",
			func() {
				m := kv.NewMapForTest(db.Noms(), "foo", `"bar"`, "hot", `"dog"`)
				c, err := db.MaybePutData(m, 0 /*lastMutationID*/)
				assert.False(c.Original.IsZeroValue())
				assert.NoError(err)
			},
			[]kv.Operation{
				{
					Op:    kv.OpAdd,
					Path:  "/foo",
					Value: []byte("\"bar\""),
				},
				{
					Op:    kv.OpAdd,
					Path:  "/hot",
					Value: []byte("\"dog\""),
				},
			},
			"",
		},
		{
			"change-2",
			func() {
				m := kv.NewMapForTest(db.Noms(), "foo", `"baz"`, "mon", `"key"`)
				c, err := db.MaybePutData(m, 0 /*lastMutationID*/)
				assert.False(c.Original.IsZeroValue())
				assert.NoError(err)
			},
			[]kv.Operation{
				{
					Op:    kv.OpReplace,
					Path:  "/foo",
					Value: []byte("\"baz\""),
				},
				{
					Op:   kv.OpRemove,
					Path: "/hot",
				},
				{
					Op:    kv.OpAdd,
					Path:  "/mon",
					Value: []byte("\"key\""),
				},
			},
			"",
		},
		{
			"no-diff",
			func() {},
			[]kv.Operation{},
			"",
		},
		{
			"fresh-non-existing-commit",
			func() {
				db, dir = LoadTempDB(assert)
				fmt.Println("newdir", dir)
				me := kv.NewMapForTest(db.Noms()).Edit()
				for _, s := range []string{"a", "b", "c"} {
					assert.NoError(me.Set(types.String(s), types.String(s)))
				}
				m := me.Build()
				c, err := db.MaybePutData(m, 0 /*lastMutationID*/)
				assert.False(c.Original.IsZeroValue())
				assert.NoError(err)
			},
			[]kv.Operation{
				kv.Operation{
					Op:   kv.OpRemove,
					Path: "/",
				},
				kv.Operation{
					Op:    kv.OpAdd,
					Path:  "/a",
					Value: json.RawMessage([]byte(`"a"`)),
				},
				kv.Operation{
					Op:    kv.OpAdd,
					Path:  "/b",
					Value: json.RawMessage([]byte(`"b"`)),
				},
				kv.Operation{
					Op:    kv.OpAdd,
					Path:  "/c",
					Value: json.RawMessage([]byte(`"c"`)),
				},
			},
			"",
		},
		{
			"fresh-empty-commit",
			func() {
				fromID = hash.Hash{}
			},
			[]kv.Operation{
				kv.Operation{
					Op:   kv.OpRemove,
					Path: "/",
				},
				kv.Operation{
					Op:    kv.OpAdd,
					Path:  "/a",
					Value: json.RawMessage([]byte(`"a"`)),
				},
				kv.Operation{
					Op:    kv.OpAdd,
					Path:  "/b",
					Value: json.RawMessage([]byte(`"b"`)),
				},
				kv.Operation{
					Op:    kv.OpAdd,
					Path:  "/c",
					Value: json.RawMessage([]byte(`"c"`)),
				},
			},
			"",
		},
		{
			"invalid-checksum",
			func() {
				m := kv.NewMapForTest(db.Noms(), "foo", `"bar"`)
				c, err := db.MaybePutData(m, 0 /*lastMutationID*/)
				assert.False(c.Original.IsZeroValue())
				assert.NoError(err)
				fromChecksum = "00000000"
			},
			[]kv.Operation{
				{
					Op:   kv.OpRemove,
					Path: "/",
				},
				{
					Op:    kv.OpAdd,
					Path:  "/foo",
					Value: []byte("\"bar\""),
				},
			},
			"",
		},
		{
			"same-commit-invalid-checksum",
			func() {
				fromChecksum = "00000000"
			},
			[]kv.Operation{
				{
					Op:   kv.OpRemove,
					Path: "/",
				},
				{
					Op:    kv.OpAdd,
					Path:  "/foo",
					Value: []byte("\"bar\""),
				},
			},
			"",
		},
		{
			"invalid-commit-id",
			func() {
				r := db.Noms().WriteValue(types.String("not a commit"))
				fromID = r.TargetHash()
			},
			nil,
			"Invalid baseStateID",
		},
	}

	for _, t := range tc {
		fromID = db.Head().Original.Hash()
		var err error
		fromChecksum = string(db.Head().Value.Checksum)
		t.f()
		c, err := kv.ChecksumFromString(fromChecksum)
		assert.NoError(err)
		r, err := db.Diff(fromID, *c, db.Head(), log.Default())
		if t.expectedError == "" {
			assert.NoError(err, t.label)
			expected, err := json.Marshal(t.expectedDiff)
			assert.NoError(err, t.label)
			actual, err := json.Marshal(r)
			assert.NoError(err, t.label)
			assert.Equal(string(expected), string(actual), t.label)
		} else {
			assert.Nil(r, t.label)
			assert.EqualError(err, t.expectedError, t.label)
		}
	}
}
