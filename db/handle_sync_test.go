package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/replicant/util/noms/jsonpatch"
)

func TestHandleSync(t *testing.T) {
	assert := assert.New(t)
	db, dir := LoadTempDB(assert)
	fmt.Println(dir)

	bundle := `function set(k, v) { db.put(k, v); }`
	var fromID hash.Hash
	tc := []struct {
		label         string
		f             func()
		expectedDiff  []jsonpatch.Operation
		expectedError string
	}{
		{
			"same-commit",
			func() {},
			[]jsonpatch.Operation{},
			"",
		},
		{
			"put-bundle",
			func() {
				db.PutBundle(types.NewBlob(db.Noms(), strings.NewReader(bundle)))
			},
			[]jsonpatch.Operation{},
			"",
		},
		{
			"change-1",
			func() {
				_, err := db.Exec("set", types.NewList(db.Noms(), types.String("foo"), types.String("bar")))
				assert.NoError(err)
				_, err = db.Exec("set", types.NewList(db.Noms(), types.String("hot"), types.String("dog")))
				assert.NoError(err)
			},
			[]jsonpatch.Operation{
				{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/foo",
					Value: []byte("\"bar\""),
				},
				{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/hot",
					Value: []byte("\"dog\""),
				},
			},
			"",
		},
		{
			"change-2",
			func() {
				ok, err := db.Del("hot")
				assert.True(ok)
				assert.NoError(err)
				_, err = db.Exec("set", types.NewList(db.Noms(), types.String("foo"), types.String("baz")))
				assert.NoError(err)
				_, err = db.Exec("set", types.NewList(db.Noms(), types.String("mon"), types.String("key")))
				assert.NoError(err)
			},
			[]jsonpatch.Operation{
				{
					Op:    jsonpatch.OpReplace,
					Path:  "/u/foo",
					Value: []byte("\"baz\""),
				},
				{
					Op:   jsonpatch.OpRemove,
					Path: "/u/hot",
				},
				{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/mon",
					Value: []byte("\"key\""),
				},
			},
			"",
		},
		{
			"no-diff",
			func() {
				err := db.Put("a", types.String("a"))
				assert.NoError(err)
				ok, err := db.Del("a")
				assert.True(ok)
				assert.NoError(err)
			},
			[]jsonpatch.Operation{},
			"",
		},
		{
			"fresh-non-existing-commit",
			func() {
				db, dir = LoadTempDB(assert)
				fmt.Println("newdir", dir)
				for _, s := range []string{"a", "b", "c"} {
					err := db.Put(s, types.String(s))
					assert.NoError(err)
				}
			},
			[]jsonpatch.Operation{
				jsonpatch.Operation{
					Op:   jsonpatch.OpRemove,
					Path: "/",
				},
				jsonpatch.Operation{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/a",
					Value: json.RawMessage([]byte(`"a"`)),
				},
				jsonpatch.Operation{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/b",
					Value: json.RawMessage([]byte(`"b"`)),
				},
				jsonpatch.Operation{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/c",
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
			[]jsonpatch.Operation{
				jsonpatch.Operation{
					Op:   jsonpatch.OpRemove,
					Path: "/",
				},
				jsonpatch.Operation{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/a",
					Value: json.RawMessage([]byte(`"a"`)),
				},
				jsonpatch.Operation{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/b",
					Value: json.RawMessage([]byte(`"b"`)),
				},
				jsonpatch.Operation{
					Op:    jsonpatch.OpAdd,
					Path:  "/u/c",
					Value: json.RawMessage([]byte(`"c"`)),
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
			"Invalid commitID",
		},
	}

	for _, t := range tc {
		fromID = db.head.Original.Hash()
		t.f()
		r, err := db.HandleSync(fromID)
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
