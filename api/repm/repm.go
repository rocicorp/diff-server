// package remp implements a Gomobile-compatible API to replicant.
package repm

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/attic-labs/noms/go/spec"

	"github.com/aboodman/replicant/cmd"
	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/chk"
)

var (
	cmds map[string]cmd.Command
)

func init() {
	cmds = map[string]cmd.Command{
		"data/put": &cmd.DataPut{}, // TODO: remove once exec exists
		"data/has": &cmd.DataHas{},
		"data/get": &cmd.DataGet{},
		"data/del": &cmd.DataDel{},
	}

}

type Connection struct {
	db *db.DB
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

func (conn *Connection) Exec(name string, cs []byte) (*Command, error) {
	rc := cmds[name]
	if rc == nil {
		return nil, fmt.Errorf("Unknown command: %s", name)
	}

	err := json.Unmarshal(cs, &rc)
	if err != nil {
		return nil, err
	}

	val := reflect.ValueOf(rc).Elem()
	inval := val.FieldByName("InStream")
	outval := val.FieldByName("OutStream")

	r := &Command{
		c:   rc,
		err: make(chan error),
	}

	if inval.IsValid() {
		inR, inW := io.Pipe()
		inval.Set(reflect.ValueOf(inR))
		r.inW = inW
	}

	if outval.IsValid() {
		outR, outW := io.Pipe()
		outval.Set(reflect.ValueOf(outW))
		r.outR = outR
	}

	go func() {
		r.err <- rc.Run(conn.db)
	}()

	return r, nil
}

type Command struct {
	c    cmd.Command
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

func (c *Command) Done() (r []byte, err error) {
	if c.inW != nil {
		err := c.inW.Close()
		chk.NoError(err)
	}
	if c.outR != nil {
		err := c.outR.Close()
		chk.NoError(err)
	}
	outVal := reflect.ValueOf(c).Elem().FieldByName("c").Elem().Elem().FieldByName("Out")
	if outVal.NumField() > 0 {
		r, err = json.Marshal(outVal.Interface())
		if err != nil {
			return nil, err
		}
	}
	return r, <-c.err
}
