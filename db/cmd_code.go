package db

import (
	"io"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
)

type CodePut struct {
	InStream io.Reader
	In       struct {
		Origin string
	}
	Out struct {
	}
}

func (c *CodePut) Run(db *DB) error {
	// TODO: Do we want to validate that it compiles or whatever???
	b := types.NewBlob(db.Noms(), c.InStream)
	err := db.PutCode(b)
	if err != nil {
		return err
	}
	commit, changes, err := db.MakeTx(c.In.Origin, types.NewEmptyBlob(db.Noms()), ".code.put", types.NewList(db.Noms(), b), datetime.Now())
	if err != nil {
		return err
	}
	if changes {
		_, err = db.Commit(commit)
		return err
	}
	return nil
}

type CodeGet struct {
	In struct {
	}
	Out struct {
		OK bool
	}
	OutStream io.Writer
}

func (c *CodeGet) Run(db *DB) error {
	b, err := db.GetCode()
	if err != nil {
		return err
	}
	c.Out.OK = true
	_, err = io.Copy(c.OutStream, b.Reader())
	if err != nil {
		return err
	}
	if wc, ok := c.OutStream.(io.WriteCloser); ok {
		err = wc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
