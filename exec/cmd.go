package exec

import (
	"errors"
	"io"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/jsoms"
)

type CodeExec struct {
	In struct {
		Origin string
		Code   jsoms.Hash
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

	var code types.Value
	var reader io.Reader
	if c.In.Name == "" {
		return errors.New("Function name parameter is required")
	}
	if isSystemFunction(c.In.Name) {
		if !c.In.Code.IsEmpty() {
			return errors.New("Invalid to specify code bundle with system function")
		}
	} else {
		if c.In.Code.IsEmpty() {
			return errors.New("Code bundle parameter required")
		}
		code = db.Noms().ReadValue(c.In.Code.Hash)
		if code == nil {
			return errors.New("Specified code bundle does not exist")
		}
		if code.Kind() != types.BlobKind {
			return errors.New("Specified code bundle hash is not a blob")
		}
		reader = code.(types.Blob).Reader()
	}

	err = Run(db, reader, c.In.Name, args)
	if err != nil {
		return err
	}

	commit, changes, err := db.MakeTx(c.In.Origin, code.(types.Blob), c.In.Name, args, datetime.Now())
	if err != nil {
		return err
	}
	if changes {
		_, err = db.Commit(commit)
		if err != nil {
			return err
		}
	}
	return nil
}
