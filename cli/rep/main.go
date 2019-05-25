package main

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/spec"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/aboodman/replicant/cmd"
	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/chk"
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

	code := app.Command("code", "Interact with code.")
	reg(sp, code, &cmd.CodePut{}, "put", "Set a new JavaScript transaction bundle.", opt{
	}, in, out, errs)
	reg(sp, code, &cmd.CodeGet{}, "get", "Get the current transaction bundle.", opt{
	}, in, out, errs)

	data := app.Command("data", "Interact with data.")
	// TODO: Remove this one once exec works.
	reg(sp, data, &cmd.DataPut{}, "put", "Write the content of stdin as a value. Value must be JSON-formatted.", opt{
		Args: []string{"ID"},
	}, in, out, errs)
	reg(sp, data, &cmd.DataHas{}, "has", "Check value existence.", opt{
		Args:     []string{"ID"},
		OutField: "OK",
	}, in, out, errs)
	reg(sp, data, &cmd.DataGet{}, "get", "Read a value.", opt{
		Args: []string{"ID"},
	}, in, out, errs)
	reg(sp, data, &cmd.DataDel{}, "del", "Delete a value.", opt{
		Args: []string{"ID"},
	}, in, out, errs)

	_, err := app.Parse(args)
	if err != nil {
		fmt.Fprintln(errs, err.Error())
		exit(1)
	}
}

func reg(sp *spec.Spec, parent *kingpin.CmdClause, rc cmd.Command, name, doc string, o opt, in io.Reader, out, errs io.Writer) {
	val := reflect.ValueOf(rc).Elem()
	inVal := val.FieldByName("In")
	outVal := val.FieldByName("Out")

	kc := parent.Command(name, doc)

	for i := 0; i < inVal.NumField(); i++ {
		fn := inVal.Type().Field(i).Name
		fv := inVal.Field(i)
		clause := kc.Arg(fn, "TODO").Required()
		// TODO: optional if pointer type
		switch fullName(fv.Type()) {
		case ".string":
			clause.StringVar(fv.Addr().Interface().(*string))
		case "github.com/attic-labs/noms/go/hash.Hash":
			kp.Hash(clause, fv.Addr().Interface().(*hash.Hash))
		default:
			panic(fmt.Sprintf("Unexpected field type: %+v", fv.Type()))
		}
	}

	rcv := reflect.ValueOf(rc).Elem()
	inStreamField := rcv.FieldByName("InStream")
	if inStreamField.IsValid() {
		chk.Equal("io.Reader", fullName(inStreamField.Type()))
		inStreamField.Set(reflect.ValueOf(in))
	}

	outStreamField := rcv.FieldByName("OutStream")
	if outStreamField.IsValid() {
		chk.Equal("io.Writer", fullName(outStreamField.Type()))
		outStreamField.Set(reflect.ValueOf(out))
	}

	kc.Action(func(_ *kingpin.ParseContext) error {
		db, err := db.Load(*sp)
		if err != nil {
			return err
		}

		err = rc.Run(db)
		if err != nil {
			return err
		}

		err = db.Commit()
		if err != nil {
			return err
		}

		if o.OutField != "" {
			if outStreamField.IsValid() {
				chk.Fail("Cannot set both OutField and OutStream")
			}
			fv := field(outVal, o.OutField)
			fmt.Fprintf(out, "%v\n", fv.Interface())
		}

		return nil
	})
}

func fullName(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}

func field(v reflect.Value, n string) reflect.Value {
	r := v.FieldByName(n)
	chk.False(r == reflect.Value{}, "Unknown field: %s", n)
	return r
}
