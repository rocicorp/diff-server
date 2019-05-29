package cmd

import (
	"io"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/db"
)

type CodePut struct {
	InStream io.Reader
	In       struct {
		Origin string
	}
	Out struct {
	}
}

func (c *CodePut) Run(db *db.DB) (err error) {
	// TODO: Do we want to validate that it compiles or whatever???
	b := types.NewBlob(db.Noms(), c.InStream)
	err = db.PutCode(b)
	db.Commit(c.In.Origin, ".code.put", types.NewList(db.Noms(), b), datetime.Now())
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
	c.Out.OK = true
	_, err = io.Copy(c.OutStream, r)
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
