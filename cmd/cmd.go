// package cmd provides a structured command-style facade to Replicant.
// This package is then used to construct all the various interfaces to Replicant, including
// Programmatic APIs for iOS, Android, C; REST server; CLI, etc.
package cmd

import (
	"io"

	"github.com/attic-labs/noms/go/hash"

	"github.com/aboodman/replicant/db"
)

type Command interface {
	Run(db db.DB) error
}

type DataPut struct {
	In struct {
		ID string
	}
	InStream io.Reader
}

func (c *DataPut) Run(db db.DB) error {
	return db.Put(c.In.ID, c.InStream)
}

type DataHas struct {
	In struct {
		ID string
	}
	Out struct {
		OK bool
	}
}

func (c *DataHas) Run(db db.DB) (err error) {
	c.Out.OK, err = db.Has(c.In.ID)
	return
}

type DataGet struct {
	In struct {
		ID string
	}
	Out struct {
		OK bool
	}
	OutStream io.Writer
}

func (c *DataGet) Run(db db.DB) (err error) {
	c.Out.OK, err = db.Get(c.In.ID, c.OutStream)
	return
}

type DataDel struct {
	In struct {
		ID string
	}
	Out struct {
		OK bool
	}
}

func (c *DataDel) Run(db db.DB) (err error) {
	c.Out.OK, err = db.Del(c.In.ID)
	return
}

type CodePut struct {
	InStream io.Reader
	Out      struct {
		Hash hash.Hash
	}
}

func (c *CodePut) Run(db db.DB) (err error) {
	c.Out.Hash, err = db.PutCode(c.InStream)
	return
}

type CodeHas struct {
	In struct {
		Hash hash.Hash
	}
	Out struct {
		OK bool
	}
}

func (c *CodeHas) Run(db db.DB) error {
	ok, err := db.HasCode(c.In.Hash)
	if err != nil {
		return err
	}
	c.Out.OK = ok
	return nil
}

type CodeGet struct {
	In struct {
		Hash hash.Hash
	}
	Out struct {
		OK bool
	}
	OutStream io.Writer
}

func (c *CodeGet) Run(db db.DB) error {
	b, err := db.GetCode(c.In.Hash)
	if err != nil {
		return err
	}
	if err == nil {
		_, err = io.Copy(c.OutStream, b.Reader())
		if err != nil {
			return err
		}
	}
	c.Out.OK = true
	return nil
}

type CodeDel struct {
	In struct {
		Hash hash.Hash
	}
	Out struct {
		OK bool
	}
}

func (c CodeDel) Run(db db.DB) (err error) {
	c.Out.OK, err = db.DelCode(c.In.Hash)
	return
}

/*
type CodeList struct {
	In struct {
	}
	Out struct {
		chan type.Blob
	}
}
func (c CodeList) Run(db db.DB) (error) {
	return errors.New("TODO")
}
*/
