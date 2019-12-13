package db

import (
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
	d, err := Load(sp, "test")
	assert.NoError(err)

	put := func(k string) {
		err = d.Put(k, types.String(k))
		assert.NoError(err)
	}

	put("")
	put("a")
	put("ba")
	put("bb")

	// TODO: limit, startAt
	tc := []struct {
		startAtID       string
		startAfterID    string
		prefix          string
		startAtIndex    uint64
		startAfterIndex uint64
		limit           int
		expected        []string
		expectedError   error
	}{
		{"", "", "a", 0, 0, 0, []string{"a"}, nil},
		{"", "", "b", 0, 0, 0, []string{"ba", "bb"}, nil},
		{"", "", "", 0, 0, 0, []string{"", "a", "ba", "bb"}, nil},
		{"", "", "", 0, 0, 2, []string{"", "a"}, nil},
		{"a", "", "", 0, 0, 0, []string{"a", "ba", "bb"}, nil},
		{"", "a", "", 0, 0, 0, []string{"ba", "bb"}, nil},
		{"", "", "ba", 0, 0, 0, []string{"ba"}, nil},
		{"", "", "bb", 0, 0, 0, []string{"bb"}, nil},
		{"", "", "", 1, 0, 0, []string{"a", "ba", "bb"}, nil},
		{"", "", "", 1, 0, 1, []string{"a"}, nil},
		{"", "", "", 0, 1, 0, []string{"ba", "bb"}, nil},
		{"", "", "", 0, 1, 1, []string{"ba"}, nil},
		{"a", "a", "", 0, 0, 0, nil, ErrConflictingStartConstraints},
		{"a", "", "a", 0, 0, 0, nil, ErrConflictingStartConstraints},
		{"a", "", "", 1, 0, 0, nil, ErrConflictingStartConstraints},
		{"a", "", "", 0, 1, 0, nil, ErrConflictingStartConstraints},
		{"", "a", "a", 0, 0, 0, nil, ErrConflictingStartConstraints},
		{"", "a", "", 1, 0, 0, nil, ErrConflictingStartConstraints},
		{"", "a", "", 0, 1, 0, nil, ErrConflictingStartConstraints},
		{"", "", "a", 1, 0, 0, nil, ErrConflictingStartConstraints},
		{"", "", "a", 0, 1, 0, nil, ErrConflictingStartConstraints},
		{"", "", "", 1, 1, 0, nil, ErrConflictingStartConstraints},
	}

	for i, t := range tc {
		msg := fmt.Sprintf("case %d, prefix: %s", i, t.prefix)
		res, err := d.Scan(exec.ScanOptions{
			StartAtID:       t.startAtID,
			StartAfterID:    t.startAfterID,
			Prefix:          t.prefix,
			StartAtIndex:    t.startAtIndex,
			StartAfterIndex: t.startAfterIndex,
			Limit:           t.limit})
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
