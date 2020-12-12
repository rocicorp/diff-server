package account

import (
	"strconv"
)

// Records is the full set of Replicache account records. These
// records come from two places for now (auto-signup accounts from
// noms and regular accounts from our hard-coded list), so it is
// important to use ReadRecords() to be sure you get them all.
type Records struct {
	NextASID uint32
	Record   map[uint32]Record // Map key is the ID.
}

// Record represents a single account record.
type Record struct {
	ID             uint32
	Name           string
	Email          string
	ClientViewURLs []string
	DateCreated    string
}

// ASIDs are issued in a separate range from regular accounts.
// See RFC: https://github.com/rocicorp/repc/issues/269
const LowestASID uint32 = 1000000

// ReadRecords returns the full set of Replicache account records. Reading
// of Records is separate from Lookup so the caller can cache Records if they
// so desire (it doesn't change very often).
func ReadRecords(db *DB) (Records, error) {
	if err := db.Reload(); err != nil {
		return Records{}, err
	}
	records := db.HeadValue()

	// records.Record is a map, which is a reference type, so we make
	// a copy of it to prevent aliasing bugs. A different API could
	// eliminate this copy, but this is an easy starting point.
	recordMap := records.Record
	records.Record = map[uint32]Record{}
	for k, v := range recordMap {
		// We'll add the regular accounts in a following step.
		if k >= LowestASID {
			records.Record[k] = v
		}
	}

	// Now overlay the hard-coded regular accounts. Yes a copy of these
	// records will have been saved to noms, but we want to use the hardcoded
	// version.
	for _, record := range regularAccounts {
		records.Record[record.ID] = record
	}

	return records, nil
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
// operation: re-read Records with ReadRecords, apply changes, and call
// WriteRecords again. Do not retry if the returned error cannot be
// converted to a RetryError (via errors.As).
func WriteRecords(db *DB, records Records) error {
	return db.SetHeadWithValue(records)
}
