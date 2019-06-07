package sync

import (
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/spec"

	"github.com/aboodman/replicant/db"
)

type Sync struct {
	In struct {
		Dest spec.Spec
	}
	Out struct {
		Merged hash.Hash
		// TODO: progress
	}
}

func (c *Sync) Run(local *db.DB) error {
	merged, err := DoSync(local, c.In.Dest)
	if err != nil {
		return err
	}
	c.Out.Merged = merged.TargetHash()
	return nil
}
