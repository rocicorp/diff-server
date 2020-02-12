package db

import (
	"io/ioutil"

	"github.com/attic-labs/noms/go/spec"
	"github.com/stretchr/testify/assert"
)

func LoadTempDB(assert *assert.Assertions) (r *DB, dir string) {
	td, err := ioutil.TempDir("", "")
	assert.NoError(err)

	sp, err := spec.ForDatabase(td)
	assert.NoError(err)

	r, err = Load(sp)
	assert.NoError(err)

	return r, td
}
