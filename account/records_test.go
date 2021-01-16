package account_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/account"
)

func TestReadAllRecordsAndLookup(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	// Add an ASID account.
	newAccount := account.Record{ID: account.LowestASID + 42, Name: "Larry"}
	accounts := db.HeadValue()
	accounts.Record[newAccount.ID] = newAccount
	assert.NoError(db.SetHeadWithValue(accounts))

	// Make sure unittest-account-adding function works.
	account.AddUnittestAccount(assert, db)

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
			accounts, err := account.ReadAllRecords(tt.db)
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

	accounts, err := account.ReadAllRecords(db)
	assert.NoError(err)
	accounts2, err := account.ReadAllRecords(db)
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

	accounts, err := account.ReadAllRecords(db)
	assert.NoError(err)
	newAccount := account.Record{ID: account.LowestASID + 42, Name: "Larry", ClientViewURLs: []string{"url"}}
	accounts.Record[newAccount.ID] = newAccount
	assert.NoError(account.WriteRecords(db, accounts))
	accounts, err = account.ReadAllRecords(db)
	assert.NoError(err)
	got, found := accounts.Record[newAccount.ID]
	assert.True(found)
	assert.Equal(newAccount.ID, got.ID)
	assert.Equal(newAccount.Name, got.Name)
	assert.Equal(newAccount.ClientViewURLs, got.ClientViewURLs)
}

func TestClientViewURLAuthorized(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name           string
		ID             uint32
		url            string
		wantAuthorized bool
		wantErr        string
		wantAdded      bool
	}{
		{
			"no such account",
			123,
			"url",
			false,
			"",
			false,
		},
		{
			"regular account, unauthorized url",
			0,
			"UNauthorized",
			false,
			"",
			false,
		},
		{
			"regular account, authorized url",
			0,
			"authorized",
			true,
			"",
			false,
		},
		{
			"auto account, authorized url",
			account.LowestASID,
			"authorized",
			true,
			"",
			false,
		},
		{
			"auto account, new url",
			account.LowestASID,
			"new should be authorized",
			true,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, dir := account.LoadTempDB(assert)
			defer func() { assert.NoError(os.RemoveAll(dir)) }()
			records := account.Records{
				0,
				map[uint32]account.Record{
					0:                  {ID: 0, ClientViewURLs: []string{"authorized"}},
					account.LowestASID: {ID: account.LowestASID, ClientViewURLs: []string{"authorized"}},
				},
			}
			assert.NoError(account.WriteRecords(db, records))
			recordsCopy := account.CopyRecords(records)

			gotAuthorized, err := account.ClientViewURLAuthorized(account.MaxASClientViewURLs, db, recordsCopy, tt.ID, tt.url)
			if tt.wantErr != "" {
				assert.Error(err)
				assert.Contains(err.Error(), tt.wantErr)
			} else {
				assert.NoError(err)
				assert.Equal(tt.wantAuthorized, gotAuthorized, tt.name)

				originalRecord, exists := records.Record[tt.ID]
				if exists {
					recordsAfter, err := account.ReadRecords(db)
					assert.NoError(err)
					urlAdded := len(originalRecord.ClientViewURLs) != len(recordsAfter.Record[tt.ID].ClientViewURLs)
					assert.Equal(tt.wantAdded, urlAdded, "%s: URLs before: %v, URLs after: %v", tt.name, originalRecord.ClientViewURLs, recordsAfter.Record[tt.ID].ClientViewURLs)
				}
			}
		})
	}
}
func TestClientViewURLAuthorizedWithMaxedURLs(t *testing.T) {
	assert := assert.New(t)
	db, dir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()
	records := account.Records{
		0,
		map[uint32]account.Record{
			account.LowestASID: {ID: account.LowestASID, ClientViewURLs: []string{}},
		},
	}
	record := records.Record[account.LowestASID]
	for i := 0; i < account.MaxASClientViewURLs; i++ {
		record.ClientViewURLs = append(record.ClientViewURLs, fmt.Sprintf("%d", i))
	}
	records.Record[account.LowestASID] = record
	assert.NoError(account.WriteRecords(db, records))

	gotAuthorized, err := account.ClientViewURLAuthorized(account.MaxASClientViewURLs, db, records, account.LowestASID, "some url")
	assert.NoError(err)
	assert.False(gotAuthorized)
}
