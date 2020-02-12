package db

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"roci.dev/replicant/exec"
)

func TestScan(t *testing.T) {
	assert := assert.New(t)
	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)
	d, err := Load(sp)
	assert.NoError(err)

	put := func(k string) {
		err = d.Put(k, types.String(k))
		assert.NoError(err)
	}

	put("")
	put("a")
	put("ba")
	put("bb")

	index := func(v int) *uint64 {
		vv := uint64(v)
		return &vv
	}

	tc := []struct {
		opts          exec.ScanOptions
		expected      []string
		expectedError error
	}{
		// no options
		{exec.ScanOptions{}, []string{"", "a", "ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{}}, []string{"", "a", "ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{}}}, []string{"", "a", "ba", "bb"}, nil},

		// prefix alone
		{exec.ScanOptions{Prefix: "a"}, []string{"a"}, nil},
		{exec.ScanOptions{Prefix: "b"}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Prefix: "b", Limit: 1}, []string{"ba"}, nil},
		{exec.ScanOptions{Prefix: "b", Limit: 100}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Prefix: "c"}, []string{}, nil},

		// start.id alone
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{Value: "a"}}}, []string{"a", "ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{Value: "a", Exclusive: true}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{Value: "aa"}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{Value: "aa", Exclusive: true}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{Value: "a"}}, Limit: 2}, []string{"a", "ba"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{Value: "bb"}}}, []string{"bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{Value: "bb", Exclusive: true}}}, []string{}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{ID: &exec.ScanID{Value: "c"}}}, []string{}, nil},

		// start.id and prefix together
		{exec.ScanOptions{Prefix: "a", Start: &exec.ScanBound{ID: &exec.ScanID{Value: "a"}}}, []string{"a"}, nil},
		{exec.ScanOptions{Prefix: "a", Start: &exec.ScanBound{ID: &exec.ScanID{Value: "b"}}}, []string{}, nil},
		{exec.ScanOptions{Prefix: "b", Start: &exec.ScanBound{ID: &exec.ScanID{Value: "a"}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Prefix: "c", Start: &exec.ScanBound{ID: &exec.ScanID{Value: "a"}}}, []string{}, nil},
		{exec.ScanOptions{Prefix: "a", Start: &exec.ScanBound{ID: &exec.ScanID{Value: "c"}}}, []string{}, nil},

		// start.index alone
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(0)}}, []string{"", "a", "ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1)}}, []string{"a", "ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1)}, Limit: 2}, []string{"a", "ba"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(4)}}, []string{}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(100)}}, []string{}, nil},

		// start.index and start.id together
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "b"}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "b", Exclusive: true}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "ba"}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "ba", Exclusive: true}}}, []string{"bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(2), ID: &exec.ScanID{Value: "a"}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(2), ID: &exec.ScanID{Value: "a", Exclusive: true}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(4), ID: &exec.ScanID{Value: "a"}}}, []string{}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "bb", Exclusive: true}}}, []string{}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "c"}}}, []string{}, nil},
		{exec.ScanOptions{Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "z"}}}, []string{}, nil},

		// prefix, start.index, and start.id together
		{exec.ScanOptions{Prefix: "b", Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "b"}}}, []string{"ba", "bb"}, nil},
		{exec.ScanOptions{Prefix: "a", Start: &exec.ScanBound{Index: index(1), ID: &exec.ScanID{Value: "b"}}}, []string{}, nil},
		{exec.ScanOptions{Prefix: "a", Start: &exec.ScanBound{Index: index(0), ID: &exec.ScanID{Value: "b"}}}, []string{}, nil},
		{exec.ScanOptions{Prefix: "a", Start: &exec.ScanBound{Index: index(0), ID: &exec.ScanID{Value: "a"}}}, []string{"a"}, nil},
		{exec.ScanOptions{Prefix: "c", Start: &exec.ScanBound{Index: index(0), ID: &exec.ScanID{Value: "a"}}}, []string{}, nil},
		{exec.ScanOptions{Prefix: "a", Start: &exec.ScanBound{Index: index(100), ID: &exec.ScanID{Value: "a"}}}, []string{}, nil},
		{exec.ScanOptions{Prefix: "a", Start: &exec.ScanBound{Index: index(0), ID: &exec.ScanID{Value: "z"}}}, []string{}, nil},
	}

	for i, t := range tc {
		js, err := json.Marshal(t.opts)
		assert.NoError(err)
		msg := fmt.Sprintf("case %d: %s", i, js)
		res, err := d.Scan(t.opts)
		if t.expectedError != nil {
			assert.Error(t.expectedError, err, msg)
			assert.Nil(res, msg)
			continue
		}
		assert.NoError(err)
		act := []string{}
		for _, it := range res {
			act = append(act, it.ID)
		}
		assert.Equal(t.expected, act, msg)
	}
}
