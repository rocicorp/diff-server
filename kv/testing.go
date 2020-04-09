package kv

import (
	"strings"

	"github.com/attic-labs/noms/go/types"
	"roci.dev/diff-server/util/chk"
	nomsjson "roci.dev/diff-server/util/noms/json"
)

// NewMap returns a new Map with the given keys and values.
func NewMapForTest(noms types.ValueReadWriter, kvs ...string) Map {
	me := NewMap(noms).Edit()
	for i := 0; i < len(kvs); i += 2 {
		v, err := nomsjson.FromJSON(strings.NewReader(kvs[i+1]), noms)
		chk.NoError(err)
		err = me.Set(types.String(kvs[i]), v)
		chk.NoError(err)
	}
	return me.Build()
}
