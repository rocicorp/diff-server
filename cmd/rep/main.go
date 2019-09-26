package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/trace"
	"strconv"
	"strings"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	jn "github.com/attic-labs/noms/go/util/json"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/aboodman/replicant/db"
	execpkg "github.com/aboodman/replicant/exec"
	servepkg "github.com/aboodman/replicant/serve"
	"github.com/aboodman/replicant/util/kp"
	rlog "github.com/aboodman/replicant/util/log"
)

const (
	dropWarning = "This command deletes an entire database and its history. This operations is not recoverable. Proceed? y/n\n"
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

	sp := kp.DatabaseSpec(app.Flag("db", "The database to connect to. Both local and remote databases are supported. For local databases, specify a directory path to store the database in. For remote databases, specify the http(s) URL to the database (usually https://replicate.to/serve/<mydb>).").Required().PlaceHolder("/path/to/db"))
	tf := app.Flag("trace", "Name of a file to write a trace to").OpenFile(os.O_RDWR|os.O_CREATE, 0644)

	var rdb *db.DB
	getDB := func() (db.DB, error) {
		if rdb != nil {
			return *rdb, nil
		}
		r, err := db.Load(*sp, "cli")
		if err != nil {
			return db.DB{}, err
		}
		rdb = r
		return *r, nil
	}
	app.Action(func(pc *kingpin.ParseContext) error {
		if pc.SelectedCommand == nil {
			return nil
		}

		// Init logging
		logOptions := rlog.Options{}
		if pc.SelectedCommand.Model().Name == "serve" {
			logOptions.Prefix = true
		}
		rlog.Init(errs, logOptions)

		if *tf != nil {
			err := trace.Start(*tf)
			if err != nil {
				return err
			}
		}

		return nil
	})
	defer func() {
		if *tf != nil {
			trace.Stop()
		}
	}()

	has(app, getDB, out)
	get(app, getDB, out)
	scan(app, getDB, out, errs)
	put(app, getDB, in)
	del(app, getDB, out)
	exec(app, getDB, out)
	sync(app, getDB)
	serve(app, sp, errs)
	drop(app, sp, in, out)

	bundle := app.Command("bundle", "Manage the currently registered bundle.")
	getBundle(bundle, getDB, out)
	putBundle(bundle, getDB, in)

	if len(args) == 0 {
		app.Usage(args)
		return
	}

	_, err := app.Parse(args)
	if err != nil {
		fmt.Fprintln(errs, err.Error())
		exit(1)
	}
}

type gdb func() (db.DB, error)

func getBundle(parent *kingpin.CmdClause, gdb gdb, out io.Writer) {
	kc := parent.Command("get", "Get the current JavaScript code bundle.")
	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}
		b, err := db.Bundle()
		if err != nil {
			return err
		}
		_, err = io.Copy(out, b.Reader())
		return err
	})
}

func putBundle(parent *kingpin.CmdClause, gdb gdb, in io.Reader) {
	kc := parent.Command("put", "Set a new JavaScript code bundle.")
	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}
		return db.PutBundle(types.NewBlob(db.Noms(), in))
	})
}

func exec(parent *kingpin.Application, gdb gdb, out io.Writer) {
	kc := parent.Command("exec", "Execute a function.")

	name := kc.Arg("name", "Name of function from current transaction bundle to execute.").Required().String()
	raw := kc.Arg("args", "JSON-formatted arguments to the function. For convenience, bare strings are also supported.").Strings()

	parse := func(s string, noms types.ValueReadWriter) (types.Value, error) {
		switch s {
		case "true":
			return types.Bool(true), nil
		case "false":
			return types.Bool(false), nil
		}
		switch s[0] {
		case '[', '{', '"':
			return jn.FromJSON(strings.NewReader(s), noms, jn.FromOptions{})
		default:
			if f, err := strconv.ParseFloat(s, 10); err == nil {
				return types.Number(f), nil
			}
		}
		return types.String(s), nil
	}

	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}

		args := make([]types.Value, 0, len(*raw))
		for _, r := range *raw {
			v, err := parse(r, db.Noms())
			if err != nil {
				return err
			}
			args = append(args, v)
		}
		output, err := db.Exec(*name, types.NewList(db.Noms(), args...))
		if output != nil {
			types.WriteEncodedValue(out, output)
		}
		return err
	})
}

func has(parent *kingpin.Application, gdb gdb, out io.Writer) {
	kc := parent.Command("has", "Check whether a value exists in the database.")
	id := kc.Arg("id", "id of the value to check for").Required().String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}
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

func get(parent *kingpin.Application, gdb gdb, out io.Writer) {
	kc := parent.Command("get", "Reads a value from the database.")
	id := kc.Arg("id", "id of the value to get").Required().String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}
		v, err := db.Get(*id)
		if err != nil {
			return err
		}
		if v == nil {
			return nil
		}
		return jn.ToJSON(v, out, jn.ToOptions{Lists: true, Maps: true, Indent: "  "})
	})
}

func scan(parent *kingpin.Application, gdb gdb, out, errs io.Writer) {
	kc := parent.Command("scan", "Scans values in-order from the database.")
	var opts execpkg.ScanOptions
	kc.Flag("prefix", "prefix of values to return").StringVar(&opts.Prefix)
	kc.Flag("start-at", "id of the value to start scanning at").StringVar(&opts.StartAtID)
	kc.Flag("start-after", "id of the value to start scanning after").StringVar(&opts.StartAfterID)
	kc.Flag("limit", "maximum number of items to return").IntVar(&opts.Limit)
	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}
		items, err := db.Scan(opts)
		if err != nil {
			fmt.Fprintln(errs, err)
			return nil
		}
		for _, it := range items {
			fmt.Fprintf(out, "%s: %s\n", it.ID, types.EncodedValue(it.Value.Value))
		}
		return nil
	})
}

func put(parent *kingpin.Application, gdb gdb, in io.Reader) {
	kc := parent.Command("put", "Reads a JSON-formated value from stdin and puts it into the database.")
	id := kc.Arg("id", "id of the value to put").Required().String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}
		v, err := jn.FromJSON(in, db.Noms(), jn.FromOptions{})
		if err != nil {
			return err
		}
		return db.Put(*id, v)
	})
}

func del(parent *kingpin.Application, gdb gdb, out io.Writer) {
	kc := parent.Command("del", "Deletes an item from the database.")
	id := kc.Arg("id", "id of the value to delete").Required().String()
	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}
		ok, err := db.Del(*id)
		if err != nil {
			return err
		}
		if !ok {
			out.Write([]byte("No such id.\n"))
		}
		return nil
	})
}

func sync(parent *kingpin.Application, gdb gdb) {
	kc := parent.Command("sync", "Sync with a replicant server.")
	remoteSpec := kp.DatabaseSpec(kc.Arg("remote", "Server to sync with. See https://github.com/attic-labs/noms/blob/master/doc/spelling.md#spelling-databases.").Required())

	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := gdb()
		if err != nil {
			return err
		}
		// TODO: progress
		return db.Sync(*remoteSpec)
	})
}

func serve(parent *kingpin.Application, sp *spec.Spec, errs io.Writer) {
	kc := parent.Command("serve", "Starts a local Replicant server.")
	port := kc.Flag("port", "The port to run on").Default("7001").Int()
	kc.Action(func(_ *kingpin.ParseContext) error {
		ps := fmt.Sprintf(":%d", *port)
		log.Printf("Listening on %s...", ps)
		s := servepkg.NewService("/", sp.DatabaseName)
		http.Handle("/", s)
		return http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	})
}

func drop(parent *kingpin.Application, sp *spec.Spec, in io.Reader, out io.Writer) {
	kc := parent.Command("drop", "Deletes a replicant database and its history.")

	r := bufio.NewReader(in)
	w := bufio.NewWriter(out)
	kc.Action(func(_ *kingpin.ParseContext) error {
		w.WriteString(dropWarning)
		w.Flush()
		answer, err := r.ReadString('\n')
		if err != nil {
			return err
		}
		answer = strings.TrimSpace(answer)
		if answer != "y" {
			return nil
		}
		noms := sp.GetDatabase()
		_, err = noms.Delete(noms.GetDataset(db.LOCAL_DATASET))
		return err
	})
}
