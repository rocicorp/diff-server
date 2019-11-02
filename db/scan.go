package db

import (
	"strings"

	"github.com/attic-labs/noms/go/types"

	"roci.dev/replicant/exec"
	"roci.dev/replicant/util/chk"
	jsnoms "roci.dev/replicant/util/noms/json"
)

const (
	defaultScanLimit = 50
)

func (db *DB) Scan(opts exec.ScanOptions) ([]exec.ScanItem, error) {
	return scan(db.head.Data(db.noms), opts)
}

func scan(data types.Map, opts exec.ScanOptions) ([]exec.ScanItem, error) {
	var st string
	if opts.StartAfterID != "" {
		st = opts.StartAfterID
	} else if opts.StartAtID != "" {
		st = opts.StartAtID
	} else {
		st = opts.Prefix
	}
	lim := opts.Limit
	if lim == 0 {
		lim = 50
	}
	it := data.IteratorFrom(types.String(st))
	res := []exec.ScanItem{}
	skippedFirst := false
	for {
		k, v := it.Next()
		chk.True((k == nil) == (v == nil), "Nilness of key and value should match")
		if k == nil {
			break
		}
		chk.True(k.Kind() == types.StringKind, "Only keys with string kinds are supported, Noms schema check should have caught this")
		ks := string(k.(types.String))
		if !skippedFirst {
			if opts.StartAfterID != "" && ks == opts.StartAfterID {
				continue
			}
			skippedFirst = true
		}
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
