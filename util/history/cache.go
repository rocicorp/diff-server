package history

import (
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/util/chk"
)

type Cache struct {
	heads hash.HashSet
	noms  types.ValueReadWriter

	// TODO: Support a window which not to look back past.
	// This requires a change to schema to have a date that is set by the server.
}

func NewCache(noms types.ValueReadWriter) *Cache {
	return &Cache{
		heads: hash.NewHashSet(),
		noms:  noms,
	}
}

func (kv *Cache) Has(h hash.Hash) bool {
	return kv.heads.Has(h)
}

func (kv *Cache) Populate(h hash.Hash) error {
	if kv.Has(h) {
		return nil
	}

	v := kv.noms.ReadValue(h)
	chk.NotNil(v, "Populate called with hash not known to Noms")

	kv.heads.Insert(h)

	var c struct {
		Parents []types.Ref
	}
	err := marshal.Unmarshal(v, &c)
	if err != nil {
		return err
	}

	for _, p := range c.Parents {
		err = kv.Populate(p.TargetHash())
		if err != nil {
			return err
		}
	}

	return nil
}
