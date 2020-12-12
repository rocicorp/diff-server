package account_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/account"
)

func TestReadRecordsAndLookup(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	// Add an ASID account.
	newAccount := account.Record{ID: account.LowestASID + 42, Name: "Larry"}
	accounts := db.HeadValue()
	accounts.Record[newAccount.ID] = newAccount
	assert.NoError(db.SetHeadWithValue(accounts))

	// Make sure unittest-account-adding function works.
	account.AddUnittestAccountWithURL(assert, db, "")

	tests := []struct {
		name      string
		db        *account.DB
		auth      string
		wantFound bool
		wantName  string
	}{
		{
			"no such account",
			db,
			"nosuchaccount",
			false,
			"",
		},
		{
			"sandbox regular account (mapped auth string)",
			db,
			"sandbox",
			true,
			"Sandbox",
		},
		{
			"unittest account (added with test helper)",
			db,
			fmt.Sprintf("%d", account.UnittestID),
			true,
			"Unittest",
		},
		{
			"sample app",
			db,
			"1",
			true,
			"Replicache Sample TODO",
		},
		{
			"new autosignup account",
			db,
			fmt.Sprintf("%d", newAccount.ID),
			true,
			"Larry",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts, err := account.ReadRecords(tt.db)
			assert.NoError(err)
			got, found := account.Lookup(accounts, tt.auth)
			assert.Equal(tt.wantFound, found, "%s", tt.name)
			if tt.wantFound {
				assert.Equal(tt.wantName, got.Name, "%s", tt.name)
			}
		})
	}
}

func TestReadRecordsDoesNotAlias(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	accounts, err := account.ReadRecords(db)
	assert.NoError(err)
	accounts2, err := account.ReadRecords(db)
	assert.NoError(err)
	newAccount := account.Record{ID: account.LowestASID + 42, Name: "Larry"}
	accounts.Record[newAccount.ID] = newAccount
	_, found := accounts2.Record[newAccount.ID]
	assert.False(found)
}

func TestWriteRecords(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	accounts, err := account.ReadRecords(db)
	assert.NoError(err)
	newAccount := account.Record{ID: account.LowestASID + 42, Name: "Larry", ClientViewURLs: []string{"url"}}
	accounts.Record[newAccount.ID] = newAccount
	assert.NoError(account.WriteRecords(db, accounts))
	accounts, err = account.ReadRecords(db)
	assert.NoError(err)
	got, found := accounts.Record[newAccount.ID]
	assert.True(found)
	assert.Equal(newAccount.ID, got.ID)
	assert.Equal(newAccount.Name, got.Name)
	assert.Equal(newAccount.ClientViewURLs, got.ClientViewURLs)
}
