// package remp implements a Gomobile-compatible API to replicant.
package repm

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/attic-labs/noms/go/spec"

	"github.com/aboodman/replicant/cmd"
	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/exec"
	"github.com/aboodman/replicant/util/chk"
	"github.com/aboodman/replicant/util/jsoms"
)

var (
	cmds map[string]cmd.Command
)

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
	rc := getCmd(name, conn.db)
	val := reflect.ValueOf(rc).Elem()
	in := val.FieldByName("In").Addr().Interface()

	if len(cs) > 0 {
		err := json.Unmarshal(cs, in)
		if err != nil {
			return nil, err
		}
	}

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

func (c *Command) Done() ([]byte, error) {
	if c.inW != nil {
		err := c.inW.Close()
		chk.NoError(err)
	}
	if c.outR != nil {
		err := c.outR.Close()
		chk.NoError(err)
	}

	rerr := <-c.err
	outVal := reflect.ValueOf(c.c).Elem().FieldByName("Out")
	var r []byte
	if outVal.NumField() > 0 {
		var err error
		r, err = json.Marshal(outVal.Interface())
		chk.NoError(err)
	}

	return r, rerr
}

func getCmd(name string, d *db.DB) cmd.Command {
	switch name {
	case "code/put":
		return &db.CodePut{}
	case "code/get":
		return &db.CodeGet{}
	case "code/run":
		currentCode, err := d.GetCode()
		chk.NoError(err)
		r := &exec.CodeExec{}
		r.In.Code = jsoms.Hash{currentCode.Hash()}
		r.In.Args = jsoms.Value{Noms: d.Noms()}
		return r
	case "data/has":
		return &db.DataHas{}
	case "data/get":
		return &db.DataGet{}
	case "data/del":
		return &db.DataDel{}
	}
	chk.Fail("Unsupported command: %s", name)
	return nil
}
