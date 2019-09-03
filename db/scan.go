package db

import (
	"strings"

	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/util/chk"
)

const (
	defaultScanLimit = 50
)

type ScanOptions struct {
	Prefix    string
	StartAtID string
	Limit     int
}

type ScanItem struct {
	ID    string      `json:"id"`
	Value types.Value `json:"value"`
}

func (db *DB) Scan(opts ScanOptions) ([]ScanItem, error) {
	data := db.head.Data(db.noms)
	st := opts.Prefix
	if st == "" {
		st = opts.StartAtID
	}
	lim := opts.Limit
	if lim == 0 {
		lim = 50
	}
	it := data.IteratorFrom(types.String(st))
	res := []ScanItem{}
	for {
		k, v := it.Next()
		chk.True((k == nil) == (v == nil), "Nilness of key and value should match")
		if k == nil {
			break
		}
		chk.True(k.Kind() == types.StringKind, "Only keys with string kinds are supported, Noms schema check should have caught this")
		ks := string(k.(types.String))
		if opts.Prefix != "" && !strings.HasPrefix(ks, opts.Prefix) {
			break
		}
		res = append(res, ScanItem{
			ID:    string(k.(types.String)),
			Value: v,
		})
		if len(res) == lim {
			break
		}
	}
	return res, nil
}
