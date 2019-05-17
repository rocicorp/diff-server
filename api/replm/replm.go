package replm

import (
	"encoding/json"
	"io"

	"github.com/attic-labs/noms/go/spec"

	"github.com/aboodman/replicant/cmd"
	"github.com/aboodman/replicant/db"
)

type Connection struct {
	db db.DB
}

func Open(dbSpec string) (*Connection, error) {
	sp, err := spec.ForDatabase(dbSpec)
	if err != nil {
		return nil, err
	}
	db, err := db.Load(sp)
	if err != nil {
		return nil, err
	}
	return &Connection{db: db}, nil
}

func (conn *Connection) Exec(cs []byte) (*Command, error) {
	var c cmd.Command
	err := json.Unmarshal(cs, &c)
	if err != nil {
		return nil, err
	}

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	r := &Command{
		inW:  inW,
		outR: outR,
		err:  make(chan error),
	}
	go func() {
		err := cmd.Dispatch(conn.db, c, inR, outW)
		outW.Close()
		r.err <- err
	}()

	return r, nil
}

type Command struct {
	inW  io.WriteCloser
	outR io.ReadCloser
	err  chan error
}

func (c *Command) Read(data []byte) (n int, err error) {
	return c.outR.Read(data)
}

func (c *Command) Write(data []byte) (n int, err error) {
	return c.inW.Write(data)
}

func (c *Command) Done() error {
	err := c.inW.Close()
	if err != nil {
		panic("Unexpected error: " + err.Error())
	}
	err = c.outR.Close()
	if err != nil {
		panic("Unexpected error: " + err.Error())
	}
	return <-c.err
}
