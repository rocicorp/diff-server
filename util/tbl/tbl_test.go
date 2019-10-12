package tbl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasics(t *testing.T) {
	assert := assert.New(t)
	tc := []struct {
		in       *Table
		expected string
	}{
		{&Table{}, ""},
		{(&Table{}).Add("foo", "bar"), "foobar\n"},
		{(&Table{}).Add("a", "a").Add("bb", "bb"), "a a\nbbbb\n"},
		{(&Table{}).Add("a", "a").Add("ccc", "ccc").Add("bb", "bb"), "a  a\ncccccc\nbb bb\n"},
	}

	for _, t := range tc {
		sb := &strings.Builder{}
		n, err := t.in.WriteTo(sb)
		assert.NoError(err)
		assert.Equal(int64(len(t.expected)), n)
		assert.Equal(t.expected, sb.String())
	}
}
