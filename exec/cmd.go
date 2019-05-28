package exec

import (
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/db"
)

type CodeExec struct {
	In struct {
		Origin string
		Name   string
		Args   types.List
	}
	Out struct {
	}
}

func (c CodeExec) Run(db *db.DB) (err error) {
	code, err := db.GetCode()
	if err != nil {
		return err
	}

	err = Run(db, code, c.In.Name, c.In.Args)
	if err != nil {
		return err
	}

	return db.Commit(c.In.Origin, c.In.Name, c.In.Args, datetime.Now())
}
