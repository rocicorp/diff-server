package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	jn "github.com/attic-labs/noms/go/util/json"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/kp"
)

type opt struct {
	Args     []string
	OutField string
}

func main() {
	impl(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Exit)
}

func impl(args []string, in io.Reader, out, errs io.Writer, exit func(int)) {
	app := kingpin.New("replicant", "Conflict-Free Replicated Database")
	app.ErrorWriter(errs)
	app.UsageWriter(errs)
	app.Terminate(exit)

	sp := kp.DatabaseSpec(app.Flag("db", "Database to connect to. See https://github.com/attic-labs/noms/blob/master/doc/spelling.md#spelling-databases.").Required().PlaceHolder("/path/to/db"))
	origin := app.Flag("origin", "The unique name of the client to use as the origin of any write transactions.").Default("cli").String()
	var rdb db.DB
	app.Action(func(_ *kingpin.ParseContext) error {
		r, err := db.Load(*sp, *origin)
		if err != nil {
			return err
		}
		rdb = *r
		return nil
	})

	code := app.Command("code", "Interact with code.")
	getBundle(code, &rdb, out)
	putBundle(code, &rdb, sp, in)
	exec(code, &rdb, sp)

	data := app.Command("data", "Interact with data.")
	has(data, &rdb, out)
	get(data, &rdb, out)
	put(data, &rdb, sp, in)

	_, err := app.Parse(args)
	if err != nil {
		fmt.Fprintln(errs, err.Error())
		exit(1)
	}
}
func getBundle(parent *kingpin.CmdClause, db *db.DB, out io.Writer) {
	kc := parent.Command("get", "Get the current JavaScript code bundle.")
	kc.Action(func(_ *kingpin.ParseContext) error {
		b, err := db.Bundle()
		if err != nil {
			return err
		}
		_, err = io.Copy(out, b.Reader())
		return err
	})
}

func putBundle(parent *kingpin.CmdClause, db *db.DB, sp *spec.Spec, in io.Reader) {
	kc := parent.Command("put", "Set a new JavaScript code bundle.")
	kc.Action(func(_ *kingpin.ParseContext) error {
		return db.PutBundle(types.NewBlob(sp.GetDatabase(), in))
	})
}

func exec(parent *kingpin.CmdClause, db *db.DB, sp *spec.Spec) {
	kc := parent.Command("run", "Execute a transaction.")

	name := kc.Arg("name", "Name of function from current transaction bundle to execute.").Required().String()
	raw := kc.Arg("args", "JSON-formatted arguments to the function. For convenience, bare strings are also supported.").Strings()

	parse := func(s string) (types.Value, error) {
		switch s {
		case "true":
			return types.Bool(true), nil
		case "false":
			return types.Bool(false), nil
		}
		switch s[0] {
		case '[', '{', '"':
			return jn.FromJSON(strings.NewReader(s), sp.GetDatabase(), jn.FromOptions{})
		default:
			if f, err := strconv.ParseFloat(s, 10); err == nil {
				return types.Number(f), nil
			}
		}
		return types.String(s), nil
	}

	kc.Action(func(_ *kingpin.ParseContext) error {
		args := make([]types.Value, 0, len(*raw))
		for _, r := range *raw {
			v, err := parse(r)
			if err != nil {
				return err
			}
			args = append(args, v)
		}
		return db.Exec(*name, types.NewList(sp.GetDatabase(), args...))
	})
}

func has(parent *kingpin.CmdClause, db *db.DB, out io.Writer) {
	kc := parent.Command("has", "Check whether a value exists in the database.")
	id := kc.Arg("id", "id of the value to check for").Required().String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		ok, err := db.Has(*id)
		if err != nil {
			return err
		}
		if ok {
			out.Write([]byte("true\n"))
		} else {
			out.Write([]byte("false\n"))
		}
		return nil
	})
}

func get(parent *kingpin.CmdClause, db *db.DB, out io.Writer) {
	kc := parent.Command("get", "Reads a value from the database.")
	id := kc.Arg("id", "id of the value to get").Required().String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		v, err := db.Get(*id)
		if err != nil {
			return err
		}
		if v == nil {
			return errors.New("not found")
		}
		return jn.ToJSON(v, out, jn.ToOptions{Lists: true, Maps: true, Indent: "  "})
	})
}

func put(parent *kingpin.CmdClause, db *db.DB, sp *spec.Spec, in io.Reader) {
	kc := parent.Command("put", "Reads a JSON-formated value from stdin and puts it into the database.")
	id := kc.Arg("id", "id of the value to put").Required().String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		v, err := jn.FromJSON(in, sp.GetDatabase(), jn.FromOptions{})
		if err != nil {
			return err
		}
		return db.Put(*id, v)
	})
}
