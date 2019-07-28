package nomsdiff

import (
	"bytes"

	"github.com/attic-labs/noms/go/diff"
	"github.com/attic-labs/noms/go/types"
)

func Diff(v1, v2 types.Value) string {
	buf := &bytes.Buffer{}
	diff.PrintDiff(buf, v1, v2, false)
	return string(buf.Bytes())
}
