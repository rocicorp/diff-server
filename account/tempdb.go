package account

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

	noms := sp.GetDatabase()
	r, err = New(noms.GetDataset("tempaccount"))
	assert.NoError(err)

	return r, td
}
