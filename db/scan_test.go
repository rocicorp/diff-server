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
		startAtID   string
		startAfterID string
		prefix   string
		limit    int
		expected []string
	}{
		{"", "", "a", 0, []string{"a"}},
		{"", "", "b", 0, []string{"ba", "bb"}},
		{"", "", "", 0, []string{"", "a", "ba", "bb"}},
		{"", "", "", 2, []string{"", "a"}},
		{"a", "", "", 0, []string{"a", "ba", "bb"}},
		{"", "a", "", 0, []string{"ba", "bb"}},
		{"a", "b", "ba", 0, []string{"ba"}},
		{"", "ba", "b", 0, []string{"bb"}},
	}

	for i, t := range tc {
		res, err := d.Scan(exec.ScanOptions{StartAtID: t.startAtID, StartAfterID: t.startAfterID, Prefix: t.prefix, Limit: t.limit})
		assert.NoError(err)
		act := []string{}
		msg := fmt.Sprintf("case %d, prefix: %s", i, t.prefix)
		for _, it := range res {
			act = append(act, it.ID)
		}
		assert.Equal(t.expected, act, msg)
	}
}
