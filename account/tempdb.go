package account

import (
	"io/ioutil"

	"github.com/attic-labs/noms/go/spec"
	"github.com/stretchr/testify/assert"
)

func LoadTempDB(assert *assert.Assertions) (r *DB, dir string) {
	td, err := ioutil.TempDir("", "")
	assert.NoError(err)

	r = LoadTempDBWithPath(assert, td)
	return r, td
}

func LoadTempDBWithPath(assert *assert.Assertions, td string) (r *DB) {
	sp, err := spec.ForDatabase(td)
	assert.NoError(err)
	
	noms := sp.GetDatabase()
	r, err = NewDB(noms.GetDataset("tempaccount"))
	assert.NoError(err)

	return r
}
