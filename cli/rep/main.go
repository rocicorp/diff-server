package main

import (
	"fmt"
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

type dir int

const (
	in dir = iota
	out
)

type opt struct {
	Flags    []string
	Args     []string
	OutField string
}

func main() {
	app := kingpin.New("replicant", "Conflict-Free Replicated Database")
	sp := kp.DatabaseSpec(app.Flag("db", "Database to connect to. See https://github.com/attic-labs/noms/blob/master/doc/spelling.md#spelling-databases.").Required().PlaceHolder("/path/to/db"))

	data := app.Command("data", "Interact with data.")
	// TODO: Remove this one once exec works.
	reg(sp, data, &cmd.DataPut{}, "put", "Write a value.", opt{
		Args: []string{"ID"},
	})
	reg(sp, data, &cmd.DataHas{}, "has", "Check value existence.", opt{
		Args:     []string{"ID"},
		OutField: "OK",
	})
	reg(sp, data, &cmd.DataGet{}, "get", "Read a value.", opt{
		Args: []string{"ID"},
	})
	reg(sp, data, &cmd.DataDel{}, "del", "Delete a value.", opt{
		Args: []string{"ID"},
	})

	code := app.Command("code", "Interact with code.")
	reg(sp, code, &cmd.CodePut{}, "put", "Write code.", opt{
		OutField: "Hash",
	})
	reg(sp, code, &cmd.CodeHas{}, "has", "Check code existence.", opt{
		Args:     []string{"Hash"},
		OutField: "OK",
	})
	reg(sp, code, &cmd.CodeGet{}, "get", "Read code.", opt{
		Args: []string{"Hash"},
	})

	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func reg(sp *spec.Spec, parent *kingpin.CmdClause, rc cmd.Command, name, doc string, o opt) {
	val := reflect.ValueOf(rc).Elem()
	inVal := val.FieldByName("In")
	outVal := val.FieldByName("Out")

	kc := parent.Command(name, doc)
	for _, _ = range o.Flags {
		// TODO
	}
	for _, fn := range o.Args {
		fv := field(inVal, fn)
		// TODO: optional if pointer type
		clause := kc.Arg(fn, "TODO").Required()
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
		inStreamField.Set(reflect.ValueOf(os.Stdin))
	}

	outStreamField := rcv.FieldByName("OutStream")
	if outStreamField.IsValid() {
		chk.Equal("io.Writer", fullName(outStreamField.Type()))
		outStreamField.Set(reflect.ValueOf(os.Stdout))
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

		if o.OutField != "" {
			if outStreamField.IsValid() {
				chk.Fail("Cannot set both OutField and OutStream")
			}
			fv := field(outVal, o.OutField)
			fmt.Printf("%v\n", fv.Interface())
		}

		err = db.Commit()
		if err != nil {
			return err
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
