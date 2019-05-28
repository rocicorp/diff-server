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

/*
steps to exec:
x rejigger cmd again to make put/get
x expose bindings to js engine
  x need fairly nice-looking apis -- requires js builtin
    x need to bake js into binary
  x across boundary can be json
  x then embed
- figure out where commit logic goes - in db?
  - commit method turns into Exec() method that commits implicitly
	- and calls out to execution engine
- cleanups:
	- db.put -> db.set
	- db should not be arg to tx functions, but in global scope
- finish cmd, test from cli and android
  - parse human-readable params, but limit to JSON
- remove `put` from cli!
*/
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
