package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	jn "github.com/attic-labs/noms/go/util/json"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/aboodman/replicant/cmd"
	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/exec"
	"github.com/aboodman/replicant/util/chk"
	"github.com/aboodman/replicant/util/jsoms"
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
	regPut(sp, code, in)
	reg(sp, code, &db.CodeGet{}, "get", "Get the current transaction bundle.", opt{}, in, out, errs)
	regExec(sp, code)

	data := app.Command("data", "Interact with data.")
	reg(sp, data, &db.DataHas{}, "has", "Check value existence.", opt{
		Args:     []string{"ID"},
		OutField: "OK",
	}, in, out, errs)
	reg(sp, data, &db.DataGet{}, "get", "Read a value.", opt{
		Args: []string{"ID"},
	}, in, out, errs)
	reg(sp, data, &db.DataDel{}, "del", "Delete a value.", opt{
		Args: []string{"ID"},
	}, in, out, errs)

	_, err := app.Parse(args)
	if err != nil {
		fmt.Fprintln(errs, err.Error())
		exit(1)
	}
}

func regPut(sp *spec.Spec, parent *kingpin.CmdClause, in io.Reader) {
	kc := parent.Command("put", "Set a new JavaScript transaction bundle.")
	rc := &db.CodePut{}
	rc.InStream = in

	kc.Flag("origin", "Name for the source of this transaction").Default("cli").StringVar(&rc.In.Origin)

	kc.Action(func(_ *kingpin.ParseContext) error {
		return runCommand(rc, sp)
	})
}

func regExec(sp *spec.Spec, parent *kingpin.CmdClause) {
	kc := parent.Command("run", "Execute a transaction.")
	rc := &exec.CodeExec{}

	kc.Flag("origin", "Name for the source of this transaction").Default("cli").StringVar(&rc.In.Origin)
	kp.Hash(kc.Flag("code", "Hash of code bundle to use - defaults to current"), &rc.In.Code.Hash)
	kc.Arg("name", "Name of function from current transaction bundle to execute").Required().StringVar(&rc.In.Name)
	raw := kc.Arg("args", "").Strings()

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
		rc.In.Args = jsoms.Value{Value: types.NewList(sp.GetDatabase(), args...)}

		if rc.In.Code.IsEmpty() {
			db, err := db.Load(*sp)
			if err != nil {
				return err
			}
			b, err := db.GetCode()
			if err != nil {
				return err
			}
			rc.In.Code = jsoms.Hash{b.Hash()}
		}

		return runCommand(rc, sp)
	})
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
		inStreamField.Set(reflect.ValueOf(in))
	}

	outStreamField := rcv.FieldByName("OutStream")
	if outStreamField.IsValid() {
		outStreamField.Set(reflect.ValueOf(out))
	}

	kc.Action(func(_ *kingpin.ParseContext) error {
		err := runCommand(rc, sp)
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

func runCommand(c cmd.Command, sp *spec.Spec) error {
	db, err := db.Load(*sp)
	if err != nil {
		return err
	}

	err = c.Run(db)
	if err != nil {
		return err
	}

	return nil
}

func fullName(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}

func field(v reflect.Value, n string) reflect.Value {
	r := v.FieldByName(n)
	chk.False(r == reflect.Value{}, "Unknown field: %s", n)
	return r
}
