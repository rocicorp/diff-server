package main

import (
	"fmt"
	"io"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/kp"
	"github.com/attic-labs/noms/go/spec"
)

type command struct {
	cmd     *kingpin.CmdClause
	handler func(st streams) error
}

type streams struct {
	In  io.Reader
	Out io.Writer
}

func main() {
	app := kingpin.New("replicant", "Conflict-Free Replicated Database")
	sp := kp.DatabaseSpec(app.Flag("db", "Database to connect to. See https://github.com/attic-labs/noms/blob/master/doc/spelling.md#spelling-databases.").Required().PlaceHolder("/path/to/db"))

	commands := []command{
		put(app, sp),
		get(app, sp),
	}

	selected := kingpin.MustParse(app.Parse(os.Args[1:]))
	for _, c := range commands {
		if selected == c.cmd.FullCommand() {
			err := c.handler(streams{os.Stdin, os.Stdout})
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				return
			}
			break
		}
	}
}

func put(app *kingpin.Application, sp *spec.Spec) (c command) {
	c.cmd = app.Command("put", "Insert or replace an object. JSON-encoded content of object should be sent to stdin.")
	id := c.cmd.Arg("id", "ID of object to write").Required().String()

	c.handler = func(st streams) error {
		db, err := db.Load(*sp)
		if err != nil {
			return err
		}
		err = db.Put(*id, st.In)
		if err != nil {
			return err
		}
		return db.Commit()
	}

	return c
}

func get(app *kingpin.Application, sp *spec.Spec) (c command) {
	c.cmd = app.Command("get", "Reads an object.")
	id := c.cmd.Arg("id", "ID of object to get").Required().String()

	c.handler = func(st streams) error {
		db, err := db.Load(*sp)
		if err != nil {
			return err
		}
		return db.Get(*id, st.Out)
	}

	return c
}
