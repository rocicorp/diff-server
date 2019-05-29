package exec

import (
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/jsoms"
)

type CodeExec struct {
	In struct {
		Origin string
		Name   string
		Args   jsoms.Value
	}
	Out struct {
	}
}

func (c *CodeExec) Run(db *db.DB) (err error) {
	var args types.List
	if c.In.Args.Value == nil {
		args = types.NewList(db.Noms())
	} else {
		args = c.In.Args.Value.(types.List)
	}

	code, err := db.GetCode()
	if err != nil {
		return err
	}

	err = Run(db, code, c.In.Name, args)
	if err != nil {
		return err
	}

	return db.Commit(c.In.Origin, c.In.Name, args, datetime.Now())
}
