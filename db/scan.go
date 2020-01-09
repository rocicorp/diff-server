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
	var it *types.MapIterator

	updateIter := func(cand *types.MapIterator) {
		if it == nil {
			it = cand
		} else if !it.Valid() {
			// the current iterator is at the end, no value could be greater
		} else if !cand.Valid() {
			// the candidate is at the end, all values are less
			it = cand
		} else if it.Key().Less(cand.Key()) {
			it = cand
		} else {
			// the current iterator is >= the candidate
		}
	}

	if opts.Prefix != "" {
		updateIter(data.IteratorFrom(types.String(opts.Prefix)))
	}

	if opts.Start != nil {
		if opts.Start.Key != nil && opts.Start.Key.Value != "" {
			sk := types.String(opts.Start.Key.Value)
			it := data.IteratorFrom(sk)
			if opts.Start.Key.Exclusive && it.Valid() && it.Key().Equals(sk) {
				it.Next()
			}
			updateIter(it)
		}
		if opts.Start.Index != nil {
			updateIter(data.IteratorAt(uint64((*opts.Start.Index))))
		}
	}

	if it == nil {
		it = data.Iterator()
	}

	lim := opts.Limit
	if lim == 0 {
		lim = 50
	}

	res := []exec.ScanItem{}
	for ; it.Valid(); it.Next() {
		k, v := it.Entry()
		chk.True(k.Kind() == types.StringKind, "Only keys with string kinds are supported, Noms schema check should have caught this")
		ks := string(k.(types.String))
		if opts.Prefix != "" && !strings.HasPrefix(ks, opts.Prefix) {
			break
		}
		res = append(res, exec.ScanItem{
			ID:    ks,
			Value: jsnoms.Make(nil, v),
		})
		if len(res) == lim {
			break
		}
	}
	return res, nil
}
