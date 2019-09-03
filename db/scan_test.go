package db

import (
	"fmt"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
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
		startAt  string
		prefix   string
		limit    int
		expected []string
	}{
		{"", "a", 0, []string{"a"}},
		{"", "b", 0, []string{"ba", "bb"}},
		{"", "", 0, []string{"", "a", "ba", "bb"}},
		{"", "", 2, []string{"", "a"}},
		{"a", "", 0, []string{"a", "ba", "bb"}},
		{"a", "b", 0, []string{"ba", "bb"}},
	}

	for i, t := range tc {
		res, err := d.Scan(ScanOptions{StartAtID: t.startAt, Prefix: t.prefix, Limit: t.limit})
		assert.NoError(err)
		act := []string{}
		msg := fmt.Sprintf("case %d, prefix: %s", i, t.prefix)
		for _, it := range res {
			act = append(act, it.ID)
		}
		assert.Equal(t.expected, act, msg)
	}
}
