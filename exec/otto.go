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

	"github.com/aboodman/replicant/util/chk"
)

const (
	cmdPut int = iota
	cmdHas
	cmdGet
	cmdDel
)

type UnknownFunctionError string

func (ufe UnknownFunctionError) Error() string {
	return fmt.Sprintf("Unknown function: %s", string(ufe))
}

type Database interface {
	Noms() types.ValueReadWriter
	Put(id string, value types.Value) error
	Has(id string) (ok bool, err error)
	Get(id string) (types.Value, error)
	Del(id string) (ok bool, err error)
}

func Run(db Database, source io.Reader, fn string, args types.List) (types.Value, error) {
	vm := o.New()

	s, err := vm.Compile("bootstrap.js", bootstrap)
	chk.NoError(err)
	_, err = vm.Run(s)
	chk.NoError(err)

	s, err = vm.Compile("bundle.js", source)
	if err != nil {
		return nil, errDetail(err)
	}

	_, err = vm.Run(s)
	if err != nil {
		return nil, errDetail(err)
	}

	vm.Set("send", func(call o.FunctionCall) o.Value {
		args := call.ArgumentList
		cmdID, err := args[0].ToInteger()
		chk.NoError(err)

		res, err := vm.Object("({})")
		chk.NoError(err)

		switch int(cmdID) {
		case cmdPut:
			v, err := jn.FromJSON(strings.NewReader(args[2].String()), db.Noms(), jn.FromOptions{})
			if err != nil {
				res.Set("error", err.Error())
				break
			}
			err = db.Put(args[1].String(), v)
			if err != nil {
				res.Set("error", err.Error())
			}

		case cmdHas:
			ok, err := db.Has(args[1].String())
			if err != nil {
				res.Set("error", err.Error())
			} else {
				res.Set("ok", ok)
			}

		case cmdGet:
			v, err := db.Get(args[1].String())
			if err != nil {
				res.Set("error", err.Error())
				break
			}
			if v == nil {
				res.Set("ok", false)
				break
			}
			sb := &strings.Builder{}
			err = jn.ToJSON(v, sb, jn.ToOptions{Lists: true, Maps: true})
			if err != nil {
				res.Set("error", err.Error())
				break
			}
			res.Set("ok", true)
			res.Set("data", sb.String())

		case cmdDel:
			ok, err := db.Del(args[1].String())
			if err != nil {
				res.Set("error", err.Error())
				break
			}
			res.Set("ok", ok)
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
		return nil, err
	}

	ov, err := f.Call(o.NullValue(), fn, string(buf.Bytes()))
	if err != nil {
		return nil, errDetail(err)
	}
	obj := ov.Object()
	okv, err := obj.Get("ok")
	chk.NoError(err)
	ok, err := okv.ToBoolean()
	chk.NoError(err)
	if !ok {
		return nil, UnknownFunctionError(fn)
	}

	res, err := obj.Get("result")
	chk.NoError(err)
	if res == o.UndefinedValue() {
		return nil, nil
	}
	r, err := jn.FromJSON(strings.NewReader(res.String()), db.Noms(), jn.FromOptions{})
	chk.NoError(err)
	return r, nil
}

func errDetail(err error) error {
	if oe, ok := err.(*o.Error); ok {
		return errors.New(oe.String())
	} else {
		return err
	}
}
