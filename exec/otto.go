package exec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/attic-labs/noms/go/types"
	jn "github.com/attic-labs/noms/go/util/json"
	o "github.com/robertkrimen/otto"

	dbp "github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/chk"
)

const (
	cmdPut int = iota
	cmdHas
	cmdGet
)

func Run(db *dbp.DB, source io.Reader, fn string, args types.List) error {
	if isSystemFunction(fn) {
		return runSystemFunction(db, fn, args)
	}

	vm := o.New()

	_, err := vm.Run(bootstrap)
	chk.NoError(err)

	_, err = vm.Run(source)
	if err != nil {
		return fmt.Errorf("Error loading code bundle: %s", err.Error())
	}

	vm.Set("send", func(call o.FunctionCall) o.Value {
		args := call.ArgumentList
		cmdID, err := args[0].ToInteger()
		chk.NoError(err)

		res, err := vm.Object("({})")
		chk.NoError(err)

		switch int(cmdID) {
		case cmdPut:
			c := dbp.DataPut{}
			c.In.ID = args[1].String()
			c.InStream = strings.NewReader(args[2].String())
			err = c.Run(db)
			if err != nil {
				res.Set("error", err.Error())
			}

		case cmdHas:
			c := dbp.DataHas{}
			c.In.ID = args[1].String()
			err = c.Run(db)
			if err != nil {
				res.Set("error", err.Error())
			} else {
				res.Set("ok", c.Out.OK)
			}

		case cmdGet:
			c := dbp.DataGet{}
			c.In.ID = args[1].String()
			sb := &strings.Builder{}
			c.OutStream = sb
			err = c.Run(db)
			if err != nil {
				res.Set("error", err.Error())
			} else {
				res.Set("ok", c.Out.OK)
				res.Set("data", sb.String())
			}
		}
		return res.Value()
	})

	f, err := vm.Get("recv")
	chk.NoError(err)
	chk.NotNil(f)

	buf := &bytes.Buffer{}
	err = jn.ToJSON(args, buf, jn.ToOptions{
		Lists: true,
		Maps:  true,
	})
	if err != nil {
		return err
	}

	_, err = f.Call(o.NullValue(), fn, string(buf.Bytes()))
	return err
}

func runSystemFunction(db *dbp.DB, fn string, args types.List) error {
	switch fn {
	case ".code.put":
		if args.Len() != 1 || args.Get(0).Kind() != types.BlobKind {
			return errors.New("Expected 1 blob argument")
		}

		// TODO: Do we want to validate that it compiles or whatever???
		return db.PutCode(args.Get(0).(types.Blob))
	default:
		chk.Fail("Unknown system function: %s", fn)
		return nil
	}
}

func isSystemFunction(fn string) bool {
	return fn[0] == '.'
}
