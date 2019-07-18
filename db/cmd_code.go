package db

import (
	"errors"
	"io"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"

	"github.com/aboodman/replicant/exec"
	"github.com/aboodman/replicant/util/chk"
	"github.com/aboodman/replicant/util/jsoms"
)

type CodePut struct {
	InStream io.Reader
	In       struct {
		Origin string
	}
	Out struct {
	}
}

func (c *CodePut) Run(db *DB) error {
	// TODO: Do we want to validate that it compiles or whatever???
	b := types.NewBlob(db.Noms(), c.InStream)
	if b.Equals(db.HeadCommit().Value.Code) {
		return nil
	}

	cc := &CodeExec{}
	cc.In.Origin = c.In.Origin
	cc.In.Name = ".code.put"
	cc.In.Args = jsoms.Value{types.NewList(db.Noms(), b), db.Noms()}

	return cc.Run(db)
}

type CodeGet struct {
	In struct {
	}
	Out struct {
		OK bool
	}
	OutStream io.Writer
}

func (c *CodeGet) Run(db *DB) error {
	b, err := db.GetCode()
	if err != nil {
		return err
	}
	c.Out.OK = true
	_, err = io.Copy(c.OutStream, b.Reader())
	if err != nil {
		return err
	}
	if wc, ok := c.OutStream.(io.WriteCloser); ok {
		err = wc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

type CodeExec struct {
	In struct {
		Origin string
		Code   jsoms.Hash
		Name   string
		Args   jsoms.Value
	}
	Out struct {
	}
}

// TODO: all this code should move into db.Exec()
func (c *CodeExec) Run(db *DB) (err error) {
	var args types.List
	if c.In.Args.Value == nil {
		args = types.NewList(db.Noms())
	} else {
		args = c.In.Args.Value.(types.List)
	}

	var codeRef types.Ref
	if c.In.Name == "" {
		return errors.New("Function name parameter is required")
	}
	ed := editor{db.Noms(), db.prevData.Edit(), db.prevCode}
	if isSystemFunction(c.In.Name) {
		if !c.In.Code.IsEmpty() {
			return errors.New("Invalid to specify code bundle with system function")
		}
		err = runSystemFunction(ed, c.In.Name, args)
	} else {
		if c.In.Code.IsEmpty() {
			return errors.New("Code bundle parameter required")
		}
		code := db.Noms().ReadValue(c.In.Code.Hash)
		if code == nil {
			return errors.New("Specified code bundle does not exist")
		}
		if code.Kind() != types.BlobKind {
			return errors.New("Specified code bundle hash is not a blob")
		}
		reader := code.(types.Blob).Reader()
		err = exec.Run(ed, reader, c.In.Name, args)
		if err != nil {
			return err
		}
		codeRef = types.NewRef(code)
	}

	newData, newCode := ed.Finalize()

	// TODO: There are all kinds of other nop checks scattered around
	// TODO: Need to handle nil?
	if newData.Equals(db.prevData) && newCode.Equals(db.prevCode) {
		return nil
	}

	// TODO: MakeTx becomes private
	commit, err := MakeTx(
		db.Noms(),
		db.HeadRef(),
		c.In.Origin,
		codeRef,
		c.In.Name,
		args,
		datetime.Now(),
		types.NewRef(newData),
		types.NewRef(newCode))
	if err != nil {
		return err
	}

	// TODO: Commit becomes private
	_, err = db.Commit(commit)
	return err
}

func runSystemFunction(ed editor, fn string, args types.List) error {
	switch fn {
	case ".code.put":
		if args.Len() != 1 || args.Get(0).Kind() != types.BlobKind {
			return errors.New("Expected 1 blob argument")
		}
		// TODO: Do we want to validate that it compiles or whatever???
		// TODO: Remove db.PutCode()
		ed.PutCode(args.Get(0).(types.Blob))
		return nil
	default:
		chk.Fail("Unknown system function: %s", fn)
		return nil
	}
}

func isSystemFunction(fn string) bool {
	return fn[0] == '.'
}
