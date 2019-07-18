package db

import (
	"github.com/attic-labs/noms/go/spec"
)

type Sync struct {
	In struct {
		Dest spec.Spec
	}
	Out struct {
		// TODO: progress
	}
}

func (c *Sync) Run(db *DB) error {
	return db.Sync(c.In.Dest)
}
