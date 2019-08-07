package memstore

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/types"
)

func New() *types.ValueStore {
	ts := &chunks.TestStorage{}
	return types.NewValueStore(ts.NewView())
}
