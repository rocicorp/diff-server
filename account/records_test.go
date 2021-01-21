package account_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/account"
	"roci.dev/diff-server/util/log"
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
	newAccount := account.Record{ID: account.LowestASID + 42, Name: "Larry", ClientViewHosts: []string{"host.com"}}
	accounts.Record[newAccount.ID] = newAccount
	assert.NoError(account.WriteRecords(db, accounts))
	accounts, err = account.ReadAllRecords(db)
	assert.NoError(err)
	got, found := accounts.Record[newAccount.ID]
	assert.True(found)
	assert.Equal(newAccount.ID, got.ID)
	assert.Equal(newAccount.Name, got.Name)
	assert.True(reflect.DeepEqual(newAccount.ClientViewHosts, got.ClientViewHosts))
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
			"http://authorized.com",
			false,
			"",
			false,
		},
		{
			"regular account, unauthorized url",
			0,
			"http://UNauthorized.com",
			false,
			"",
			false,
		},
		{
			"regular account, authorized url",
			0,
			"http://authorized.com",
			true,
			"",
			false,
		},
		{
			"regular account, authorized url includes port",
			0,
			"http://authorized.com:1234/somepath",
			true,
			"",
			false,
		},
		{
			"auto account, authorized url",
			account.LowestASID,
			"http://authorized.com",
			true,
			"",
			false,
		},
		{
			"auto account, new url",
			account.LowestASID,
			"http://newhost.shouldbeauthorized.com",
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
					0:                  {ID: 0, ClientViewHosts: []string{"authorized.com"}},
					account.LowestASID: {ID: account.LowestASID, ClientViewHosts: []string{"authorized.com"}},
				},
			}
			assert.NoError(account.WriteRecords(db, records))
			recordsCopy := account.CopyRecords(records)

			gotAuthorized, err := account.ClientViewURLAuthorized(account.MaxASClientViewHosts, db, recordsCopy, tt.ID, tt.url, log.Default())
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
					urlAdded := len(originalRecord.ClientViewHosts) != len(recordsAfter.Record[tt.ID].ClientViewHosts)
					assert.Equal(tt.wantAdded, urlAdded, "%s: URLs before: %v, URLs after: %v", tt.name, originalRecord.ClientViewHosts, recordsAfter.Record[tt.ID].ClientViewHosts)
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
			account.LowestASID: {ID: account.LowestASID, ClientViewHosts: []string{}},
		},
	}
	record := records.Record[account.LowestASID]
	for i := 0; i < account.MaxASClientViewHosts; i++ {
		record.ClientViewHosts = append(record.ClientViewHosts, fmt.Sprintf("%d.com", i))
	}
	records.Record[account.LowestASID] = record
	assert.NoError(account.WriteRecords(db, records))

	gotAuthorized, err := account.ClientViewURLAuthorized(account.MaxASClientViewHosts, db, records, account.LowestASID, "http://somenewhost.com", log.Default())
	assert.NoError(err)
	assert.False(gotAuthorized)
}

func TestCopyRecord(t *testing.T) {
	assert := assert.New(t)

	record := account.Record{
		ID:              1,
		Name:            "name",
		Email:           "email",
		ClientViewHosts: []string{"host1"},
		DateCreated:     "date",
		ClientViewURLs:  []string{"url1"},
	}
	copy := account.CopyRecord(record)
	assert.True(reflect.DeepEqual(record, copy))

	// Ensure no aliasing.
	copy.ClientViewHosts = append(copy.ClientViewHosts, "host2")
	assert.NotEqual(len(record.ClientViewHosts), len(copy.ClientViewHosts))
	copy.ClientViewURLs = append(copy.ClientViewURLs, "url2")
	assert.NotEqual(len(record.ClientViewURLs), len(copy.ClientViewURLs))
}
