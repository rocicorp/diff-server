package cmd

import (
	"io"

	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/db"
)

type CodePut struct {
	InStream io.Reader
	In struct {
	}
	Out struct {
	}
}

func (c *CodePut) Run(db *db.DB) (err error) {
	err = db.PutCode(c.InStream)
	return
}

type CodeGet struct {
	In struct {
	}
	Out struct {
		OK bool
	}
	OutStream io.Writer
}

func (c *CodeGet) Run(db *db.DB) error {
	r, err := db.GetCode()
	if err != nil {
		return err
	}
	_, err = io.Copy(c.OutStream, r)
	if err != nil {
		return err
	}
	return nil
}

type CodeExec struct {
	In struct {
		Name string
		Args types.List
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
- finish cmd, test from cli and android
  - parse human-readable params, but limit to JSON
- remove `put` from cli!
*/
func (c CodeExec) Run(db *db.DB) (err error) {
	return nil
}
