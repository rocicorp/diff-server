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

func AddUnittestAccount(assert *assert.Assertions, db *DB) {
	accounts, err := ReadAllRecords(db)
	assert.NoError(err)
	record := Record{ID: UnittestID, Name: "Unittest", ClientViewHosts: []string{}}
	accounts.Record[record.ID] = record
	assert.NoError(WriteRecords(db, accounts))
}

func AddUnittestAccountHost(assert *assert.Assertions, db *DB, host string) {
	accounts, err := ReadAllRecords(db)
	assert.NoError(err)
	record, exists := accounts.Record[UnittestID]
	assert.True(exists)
	record.ClientViewHosts = append(record.ClientViewHosts, host)
	accounts.Record[UnittestID] = record
	assert.NoError(WriteRecords(db, accounts))
}

func AddUnittestAccountURL(assert *assert.Assertions, db *DB, url string) {
	accounts, err := ReadAllRecords(db)
	assert.NoError(err)
	record, exists := accounts.Record[UnittestID]
	assert.True(exists)
	record.ClientViewURLs = append(record.ClientViewURLs, url)
	accounts.Record[UnittestID] = record
	assert.NoError(WriteRecords(db, accounts))
}
