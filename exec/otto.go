package exec

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/attic-labs/noms/go/types"
	jn "github.com/attic-labs/noms/go/util/json"
	o "github.com/robertkrimen/otto"

	"github.com/aboodman/replicant/cmd"
	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/chk"
)

const (
	cmdPut int = iota
	cmdHas
	cmdGet
)

func Run(db *db.DB, source io.Reader, fn string, args types.List) error {
	vm := o.New()

	_, err := vm.Run(bootstrap)
	chk.NoError(err)

	_, err = vm.Eval(source)
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
			c := cmd.DataPut{}
			c.In.ID = args[1].String()
			c.InStream = strings.NewReader(args[2].String())
			err = c.Run(db)
			if err != nil {
				res.Set("error", err.Error())
			}

		case cmdHas:
			c := cmd.DataHas{}
			c.In.ID = args[1].String()
			err = c.Run(db)
			if err != nil {
				res.Set("error", err.Error())
			} else {
				res.Set("ok", c.Out.OK)
			}

		case cmdGet:
			c := cmd.DataGet{}
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
