package account

import (
	"strconv"
)

// Records contains the set of Replicache account records (all of them
// if read with ReadAllRecords, just those from the DB if read with
// ReadRecords).
type Records struct {
	NextASID uint32
	Record   map[uint32]Record // Map key is the ID.
}

// CopyRecords deep copies Records (it contains a pointer type).
func CopyRecords(records Records) Records {
	copy := Records{
		NextASID: records.NextASID,
		Record:   make(map[uint32]Record, len(records.Record)),
	}
	for _, record := range records.Record {
		copy.Record[record.ID] = CopyRecord(record)
	}
	return copy
}

// Record represents a single account record.
type Record struct {
	ID             uint32
	Name           string
	Email          string
	ClientViewURLs []string
	DateCreated    string
}

// CopyRecord deep copies a Record (it contains a pointer type).
func CopyRecord(record Record) Record {
	copy := Record{
		ID:             record.ID,
		Name:           record.Name,
		Email:          record.Email,
		ClientViewURLs: make([]string, 0, len(record.ClientViewURLs)),
		DateCreated:    record.DateCreated,
	}
	for _, url := range record.ClientViewURLs {
		copy.ClientViewURLs = append(copy.ClientViewURLs, url)
	}
	return copy
}

// ASIDs are issued in a separate range from regular accounts.
// See RFC: https://github.com/rocicorp/repc/issues/269
const LowestASID uint32 = 1000000

// We limit the number of auto-added client view URLs for auto-signup accounts.
const MaxASClientViewURLs int = 10

// ReadAllRecords returns the full set of Replicache account records. Reading
// of Records is separate from Lookup so the caller can cache Records if they
// so desire (it doesn't change very often).
func ReadAllRecords(db *DB) (Records, error) {
	dbRecords, err := ReadRecords(db)
	if err != nil {
		return Records{}, err
	}

	// Now overlay the hard-coded regular accounts, removing any stale regular
	// account records that might have been saved. Since we are mutating records
	// we make a copy of it first :( Otherwise others who have a handle on it
	// will see our changes.
	//
	// And yes ugh: records are iterated in random order so this iterates
	// ALL our account records.
	records := CopyRecords(dbRecords)
	for _, record := range records.Record {
		if record.ID < LowestASID {
			delete(records.Record, record.ID)
		}
	}
	for _, record := range RegularAccounts {
		records.Record[record.ID] = record
	}

	return records, nil
}

// ReadRecords reads records from the db WITHOUT overlaying the production
// account records.
func ReadRecords(db *DB) (Records, error) {
	if err := db.Reload(); err != nil {
		return Records{}, err
	}
	return db.HeadValue(), nil
}

// Lookup returns the account record for the given authorization string
// and true, or the empty Record and false if it does not exist.
func Lookup(records Records, authorization string) (Record, bool) {
	// We have a special-case account where we send an auth string
	// instead of an ID in the Authorization header, so here do the
	// mapping manually. We could clean this up if we wanted, but it's
	// still not a bad idea to have indirection between the Authorization
	// header and an account Record eg if we wanted to hand out actual
	// authorization tokens that would need to be decoded and validated.
	if authorization == "sandbox" {
		authorization = "0"
	}
	id, err := strconv.ParseUint(authorization, 10, 32)
	if err != nil {
		return Record{}, false
	}
	r, found := records.Record[uint32(id)]
	return r, found
}

// WriteRecords writes the given records to the underlying db. It might
// return an RetryError in which case the caller should retry the entire
// operation: re-read Records with ReadRecords, copy it, apply changes,
// and call WriteRecords again. Do not retry if the returned error cannot be
// converted to a RetryError (via errors.As).
func WriteRecords(db *DB, records Records) error {
	return db.SetHeadWithValue(records)
}

// ClientViewURLAuthorized returns a bool indicating whether the URL the client
// is attempting to fetch from is authorized. We allow auto-signup accounts to
// fetch their client view from any URL up to some number of unique URLs. We
// limit this number to prevent spamming and require fixed, explicitly configured
// URLs for the non-ASID case for security.
//
// ClientViewURLAuthorized assumes that records is mutable. If the caller doesn't
// want to see changes from ClientViewURLAuthorized it should pass in a copy from
// CopyRecords().
func ClientViewURLAuthorized(maxASClientViewURLs int, db *DB, records Records, ID uint32, url string) (bool, error) {
	record, exists := records.Record[ID]
	if !exists {
		return false, nil
	}

	for _, authorizedURL := range record.ClientViewURLs {
		if url == authorizedURL {
			return true, nil
		}
	}
	// Regular accounts have a fixed list of authorized URLs.
	if !isASID(record.ID) {
		return false, nil
	}

	// Here we know this is an auto-signup account and the url is not in the list.
	if len(record.ClientViewURLs) >= maxASClientViewURLs {
		return false, nil
	}

	record.ClientViewURLs = append(record.ClientViewURLs, url)
	records.Record[record.ID] = record
	// TODO retry
	if err := WriteRecords(db, records); err != nil {
		return false, err
	}
	return true, nil
}

func isASID(id uint32) bool {
	return id >= LowestASID
}
