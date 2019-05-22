package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/aboodman/replicant/cmd"
	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/kp"
)

func main() {
	app := kingpin.New("replicant", "Conflict-Free Replicated Database")
	sp := kp.DatabaseSpec(app.Flag("db", "Database to connect to. See https://github.com/attic-labs/noms/blob/master/doc/spelling.md#spelling-databases.").Required().PlaceHolder("/path/to/db"))

	var command = cmd.Command{}

	ct := reflect.TypeOf(command)
	for i := 0; i < ct.NumField(); i++ {
		addSubCommand(app, &command, ct, i)
	}

	_, err := app.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	db, err := db.Load(*sp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	err = cmd.DispatchSync(db, command, os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	db.Commit()
}

func addSubCommand(app *kingpin.Application, cmd *cmd.Command, cmdType reflect.Type, fieldIndex int) {
	cmdVal := reflect.ValueOf(cmd)
	subCmdField := cmdType.Field(fieldIndex)
	subCmdVal := reflect.New(subCmdField.Type.Elem())
	kingpinCommand := app.Command(strings.ToLower(subCmdField.Name), "TODO")
	kingpinCommand.PreAction(func(*kingpin.ParseContext) error {
		cmdVal.Elem().Field(fieldIndex).Set(subCmdVal)
		return nil
	})
	for i := 0; i < subCmdField.Type.Elem().NumField(); i++ {
		ft := subCmdField.Type.Elem().Field(i)
		fv := subCmdVal.Elem().Field(i)
		clause := kingpinCommand.Arg(strings.ToLower(ft.Name), "TODO").Required()
		switch ft.Type {
		case reflect.TypeOf(""):
			clause.StringVar(fv.Addr().Interface().(*string))
		default:
			panic(fmt.Sprintf("Unexpected field type: %+v", ft.Type))
		}
	}
}
