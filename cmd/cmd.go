package cmd

import (
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

func Dispatch(db db.DB, cmd Command, in io.Reader, out io.Writer) error {
	switch {
	case cmd.Put != nil:
		return db.Put(string(cmd.Put.ID), in)
	case cmd.Get != nil:
		return db.Get(string(cmd.Get.ID), out)
	}
	return errors.New("Unknown command")
}
