// Package exec provides the ability to execute JavaScript against a Replicant database, as
// when transactions are executed.
package exec

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/attic-labs/noms/go/types"
	o "github.com/robertkrimen/otto"

	"github.com/aboodman/replicant/util/chk"
	jsnoms "github.com/aboodman/replicant/util/noms/json"
)

const (
	cmdPut int = iota
	cmdHas
	cmdGet
	cmdDel
	cmdScan
)

type UnknownFunctionError string

func (ufe UnknownFunctionError) Error() string {
	return fmt.Sprintf("Unknown function: %s", string(ufe))
}

type ScanOptions struct {
	Prefix       string `json:"prefix,omitempty"`
	StartAtID    string `json:"startAtID,omitempty"`
	StartAfterID string `json:"startAfterID,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	// Future: EndAtID, EndBeforeID
}

type ScanItem struct {
	ID    string       `json:"id"`
	Value jsnoms.Value `json:"value"`
}

type Database interface {
	Noms() types.ValueReadWriter
	Put(id string, value types.Value) error
	Has(id string) (ok bool, err error)
	Get(id string) (types.Value, error)
	Scan(opts ScanOptions) ([]ScanItem, error)
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
			v := jsnoms.Make(db.Noms(), nil)
			json.NewDecoder(strings.NewReader(args[2].String())).Decode(&v)
			if err != nil {
				res.Set("error", err.Error())
				break
			}
			err = db.Put(args[1].String(), v.Value)
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
			err = json.NewEncoder(sb).Encode(jsnoms.Make(db.Noms(), v))
			if err != nil {
				res.Set("error", err.Error())
				break
			}
			res.Set("ok", true)
			res.Set("data", sb.String())

		case cmdScan:
			var opts ScanOptions
			err := json.NewDecoder(strings.NewReader(args[1].String())).Decode(&opts)
			if err != nil {
				res.Set("error", err.Error())
				break
			}
			r, err := db.Scan(opts)
			if err != nil {
				res.Set("error", err.Error())
				break
			}
			sb := &strings.Builder{}
			err = json.NewEncoder(sb).Encode(r)
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

	sb := &strings.Builder{}
	err = json.NewEncoder(sb).Encode(jsnoms.MakeList(db.Noms(), args))
	if err != nil {
		return nil, err
	}

	ov, err := f.Call(o.NullValue(), fn, sb.String())
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
	r := jsnoms.Make(db.Noms(), nil)
	err = json.NewDecoder(strings.NewReader(res.String())).Decode(&r)
	chk.NoError(err)
	return r.Value, nil
}

func errDetail(err error) error {
	if oe, ok := err.(*o.Error); ok {
		return errors.New(oe.String())
	} else {
		return err
	}
}
