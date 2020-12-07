package account_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/account"
)

func TestInit(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer assert.NoError(os.RemoveAll(dir))

	assert.Equal(account.LowestASID, db.HeadValue().NextASID)
	assert.Equal(0, len(db.HeadValue().AutoSignup))
}

func TestReload(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer assert.NoError(os.RemoveAll(dir))

	accounts := db.HeadValue()
	accounts.NextASID = 1
	accounts.AutoSignup[1] = account.ASAccount{ASID: 1}

	assert.NoError(db.Reload())
	accounts = db.HeadValue()
	assert.Equal(0, accounts.NextASID)
	assert.Nil(accounts.AutoSignup[1])
}

func TestSetHead(t *testing.T) {
	//assert := assert.New(t)
	//db, dir := account.LoadTempDB(assert)
	//defer assert.NoError(os.RemoveAll(dir))

	//accounts1 := jj
}
