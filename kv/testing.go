package kv

import (
	"github.com/attic-labs/noms/go/types"
	"roci.dev/diff-server/util/chk"
)

// NewMap returns a new Map with the given keys and values.
func NewMapForTest(noms types.ValueReadWriter, kvs ...string) Map {
	me := NewMap(noms).Edit()
	for i := 0; i < len(kvs); i += 2 {
		err := me.Set(kvs[i], []byte(kvs[i+1]))
		chk.NoError(err)
	}
	return me.Build()
}
