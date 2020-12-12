package account

import (
	"io/ioutil"

	"github.com/stretchr/testify/assert"
)

func LoadTempDB(assert *assert.Assertions) (r *DB, dir string) {
	td, err := ioutil.TempDir("", "")
	assert.NoError(err)

	r = LoadTempDBWithPath(assert, td)
	return r, td
}

func LoadTempDBWithPath(assert *assert.Assertions, td string) (r *DB) {
	r, err := NewDB(td)
	assert.NoError(err)
	return r
}

const UnittestID = 0xFFFFFFFF

func AddUnittestAccountWithURL(assert *assert.Assertions, db *DB, clientViewURL string) {
	accounts, err := ReadRecords(db)
	assert.NoError(err)
	record := Record{ID: UnittestID, Name: "Unittest", ClientViewURLs: []string{clientViewURL}}
	accounts.Record[record.ID] = record
	assert.NoError(WriteRecords(db, accounts))
}
