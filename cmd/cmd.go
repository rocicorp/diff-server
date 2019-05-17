package cmd

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/aboodman/replicant/db"
)

type Command struct {
	Put *Put `json:"put,omitempty"`
	Get *Get `json:"get,omitempty"`
}

type Put struct {
	ID string `json:"id"`
}

type Get struct {
	ID string `json:"id"`
}

func DispatchString(db db.DB, cs []byte) (outR io.ReadCloser, inW io.WriteCloser, ec chan error, err error) {
	var c Command
	err = json.Unmarshal(cs, &c)
	if err != nil {
		return nil, nil, nil, err
	}
	outR, inW, ec = Dispatch(db, c)
	return
}

func Dispatch(db db.DB, cmd Command) (outR io.ReadCloser, inW io.WriteCloser, ec chan error) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ec = make(chan error)

	go func() {
		err := DispatchSync(db, cmd, inR, outW)
		outW.Close()
		ec <- err
	}()

	return outR, inW, ec
}

func DispatchSync(db db.DB, cmd Command, inR io.Reader, outW io.Writer) error {
	switch {
	case cmd.Put != nil:
		return db.Put(string(cmd.Put.ID), inR)
	case cmd.Get != nil:
		return db.Get(string(cmd.Get.ID), outW)
	}
	return errors.New("Unknown command")
}
