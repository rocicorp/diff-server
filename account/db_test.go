package account_test

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/account"
)

func TestInit(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	assert.Equal(account.LowestASID, db.HeadValue().NextASID)
	assert.Equal(0, len(db.HeadValue().Record))
}

func TestReload(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	// Use db2 to change head behind db's back.
	db2 := account.LoadTempDBWithPath(assert, dir)
	accounts := db2.HeadValue()
	accounts.NextASID = 2
	accounts.Record[2] = account.Record{ID: 2}
	assert.NoError(db2.SetHeadWithValue(accounts))

	// Now ensure that if we reload db we see the changes from db2.
	assert.NoError(db.Reload())
	assert.Equal(uint32(2), db.HeadValue().NextASID)
	_, exists := db.HeadValue().Record[2]
	assert.True(exists)
}

func TestSetHead(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	accounts := db.HeadValue()
	accounts.NextASID = 123
	accounts.Record[123] = account.Record{ID: 123}
	assert.NoError(db.SetHeadWithValue(accounts))

	gotDB := account.LoadTempDBWithPath(assert, dir)
	assert.Equal(accounts, gotDB.HeadValue())
}

func TestConcurrentSetHead(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	// Set head behind db's back.
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func() {
		otherDB := account.LoadTempDBWithPath(assert, dir)
		accounts := otherDB.HeadValue()
		accounts.NextASID = 1
		accounts.Record[1] = account.Record{ID: 1}
		err = otherDB.SetHeadWithValue(accounts)
		wg.Done()
	}()
	wg.Wait()
	assert.NoError(err)

	// Head has been set behind our back so we expect to get a RetryError
	// when we try to set head.
	accounts := db.HeadValue()
	accounts.NextASID = 2
	accounts.Record[2] = account.Record{ID: 2}
	err = db.SetHeadWithValue(accounts)
	assert.Error(err)
	var retryError account.RetryError
	assert.True(errors.As(err, &retryError))
}
