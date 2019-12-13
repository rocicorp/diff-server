package db

import (
	"errors"
	"strings"

	"github.com/attic-labs/noms/go/types"

	"roci.dev/replicant/exec"
	"roci.dev/replicant/util/chk"
	jsnoms "roci.dev/replicant/util/noms/json"
)

const (
	defaultScanLimit = 50
)

var (
	ErrConflictingStartConstraints = errors.New("Only one of the startAtID, startAfterID, startAtIndex, startAfterIndex, and prefix fields may be present")
)

func (db *DB) Scan(opts exec.ScanOptions) ([]exec.ScanItem, error) {
	return scan(db.head.Data(db.noms), opts)
}

func scan(data types.Map, opts exec.ScanOptions) ([]exec.ScanItem, error) {
	var startID string
	var startIndex uint64
	startFields := 0
	skipNext := false

	if opts.StartAtID != "" {
		startID = opts.StartAtID
		startFields++
	}
	if opts.StartAfterID != "" {
		startID = opts.StartAfterID
		startFields++
		skipNext = true
	}
	if opts.Prefix != "" {
		startID = opts.Prefix
		startFields++
	}
	if opts.StartAtIndex > 0 {
		startIndex = opts.StartAtIndex
		startFields++
	}
	if opts.StartAfterIndex > 0 {
		startIndex = opts.StartAfterIndex
		startFields++
		skipNext = true
	}

	if startFields > 1 {
		return nil, ErrConflictingStartConstraints
	}

	lim := opts.Limit
	if lim == 0 {
		lim = 50
	}

	var it types.MapIterator
	if startID != "" {
		it = data.IteratorFrom(types.String(startID))
	} else {
		it = data.IteratorAt(startIndex)
	}

	res := []exec.ScanItem{}
	for {
		k, v := it.Next()
		chk.True((k == nil) == (v == nil), "Nilness of key and value should match")
		if k == nil {
			break
		}
		chk.True(k.Kind() == types.StringKind, "Only keys with string kinds are supported, Noms schema check should have caught this")
		if skipNext {
			skipNext = false
			continue
		}
		ks := string(k.(types.String))
		if opts.Prefix != "" && !strings.HasPrefix(ks, opts.Prefix) {
			break
		}
		res = append(res, exec.ScanItem{
			ID:    string(k.(types.String)),
			Value: jsnoms.Make(nil, v),
		})
		if len(res) == lim {
			break
		}
	}
	return res, nil
}
