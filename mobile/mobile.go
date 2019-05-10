package mobile

import (
	"io"

	"github.com/aboodman/replicant/db"
	"github.com/attic-labs/noms/go/spec"
)

type Database struct {
	db db.DB
}

type Reader io.Reader
type Writer io.Writer

func Load(sp string) (*Database, error) {
	s, err := spec.ForDatabase(sp)
	if err != nil {
		return &Database{}, err
	}
	dbi, err := db.Load(s)
	if err != nil {
		return &Database{}, err
	}
	return &Database{dbi}, nil
}

func (db Database) Put(id string, r Reader) error {
	return db.db.Put(id, io.Reader(r))
}

func (db Database) Get(id string, w Writer) error {
	return db.db.Get(id, w)
}

func (db Database) Commit() error {
	return db.db.Commit()
}
