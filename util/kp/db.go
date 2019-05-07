package kp

import (
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type DatabaseValue struct {
	db *datas.Database
}

func (db DatabaseValue) Set(value string) error {
	sp, err := spec.ForDatabase(value)
	if err != nil {
		return err
	}
	t := sp.GetDatabase()
	db.db = &t
	return nil
}

func (db DatabaseValue) String() string {
	return ""
}

func Database(s kingpin.Settings) (target *datas.Database) {
	s.SetValue(DatabaseValue{target})
	return
}
