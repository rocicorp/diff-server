package cmd

import (
	"io"

	"github.com/aboodman/replicant/db"
)

type DataPut struct {
	In struct {
		ID   string
	}
	InStream io.Reader
	Out struct {
	}
}

func (c *DataPut) Run(db *db.DB) error {
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

func (c *DataHas) Run(db *db.DB) (err error) {
	c.Out.OK, err = db.Has(c.In.ID)
	return
}

type DataGet struct {
	In struct {
		ID string
	}
	Out struct {
		OK   bool
	}
	OutStream io.Writer
}

func (c *DataGet) Run(db *db.DB) (err error) {
	c.Out.OK, err = db.Get(c.In.ID, c.OutStream)
	if err != nil {
		return err
	}
	return nil
}

type DataDel struct {
	In struct {
		ID string
	}
	Out struct {
		OK bool
	}
}

func (c *DataDel) Run(db *db.DB) (err error) {
	c.Out.OK, err = db.Del(c.In.ID)
	return
}
