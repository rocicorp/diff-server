package reachable

import (
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/util/chk"
)

type Set struct {
	heads hash.HashSet
	noms  types.ValueReadWriter

	// TODO: Support a window which not to look back past.
	// This requires a change to schema to have a date that is set by the server.
}

func New(noms types.ValueReadWriter) *Set {
	return &Set{
		heads: hash.NewHashSet(),
		noms:  noms,
	}
}

func (s *Set) Has(h hash.Hash) bool {
	return s.heads.Has(h)
}

func (s *Set) Populate(h hash.Hash) error {
	if s.Has(h) {
		return nil
	}

	v := s.noms.ReadValue(h)
	chk.NotNil(v, "Populate called with hash not known to Noms")

	s.heads.Insert(h)

	var c struct {
		Parents []types.Ref
	}
	err := marshal.Unmarshal(v, &c)
	if err != nil {
		return err
	}

	for _, p := range c.Parents {
		err = s.Populate(p.TargetHash())
		if err != nil {
			return err
		}
	}

	return nil
}
