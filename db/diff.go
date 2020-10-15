package db

import (
	"fmt"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	zl "github.com/rs/zerolog"

	"roci.dev/diff-server/kv"
)

func fullSync(version uint32, db *DB, from hash.Hash, l zl.Logger) ([]kv.Operation, Commit) {
	l.Debug().Msgf("Requested sync %s basis could not be found - sending a full sync", from.String())

	var op kv.Operation
	if version < 2 {
		op = kv.Operation{
			Op:   kv.OpRemove,
			Path: "/",
		}
	} else {
		op =
			kv.Operation{
				Op:          kv.OpReplace,
				Path:        "",
				ValueString: "{}",
			}

	}
	m := kv.NewMap(db.Noms())
	return []kv.Operation{op}, makeCommit(db.Noms(), types.Ref{}, datetime.Epoch, db.ds.Database().WriteValue(m.NomsMap()), m.NomsChecksum(), 0 /*lastMutationID*/)
}

func maybeDecodeCommit(v types.Value, h hash.Hash, expectedChecksum kv.Checksum, l zl.Logger) (Commit, error) {
	var c Commit
	err := marshal.Unmarshal(v, &c)
	if err != nil {
		return Commit{}, fmt.Errorf("could not decode basis %s: %w", h, err)
	}
	checksum, err := kv.ChecksumFromString(string(c.Value.Checksum))
	if err != nil {
		return Commit{}, fmt.Errorf("couldn't parse checksum from basis %s: %s", h, string(c.Value.Checksum))
	}
	if !checksum.Equal(expectedChecksum) {
		return Commit{}, fmt.Errorf("checksum mismatch: %s from client, %s in db", expectedChecksum, checksum)
	}
	return c, nil
}

func (db *DB) Diff(version uint32, fromHash hash.Hash, fromChecksum kv.Checksum, to Commit, l zl.Logger) ([]kv.Operation, error) {
	r := []kv.Operation{}
	var fc Commit
	var err error
	v := db.Noms().ReadValue(fromHash)
	if v == nil {
		// Unknown basis is not an error: maybe it's really old or we're starting up cold.
		l.Info().Msgf("Sending full sync: unknown basis %s", fromHash)
		r, fc = fullSync(version, db, fromHash, l)
	} else {
		fc, err = maybeDecodeCommit(v, fromHash, fromChecksum, l)
		if err != nil {
			// Inability to decode a Commit or getting the wrong checksum is an error.
			l.Error().Msgf("Sending full sync: cannot diff from basis %s: %s", fromHash, err)
			r, fc = fullSync(version, db, fromHash, l)
		}
	}

	if !fc.Value.Data.Equals(to.Value.Data) {
		fm := fc.Data(db.Noms())
		tm := to.Data(db.Noms())
		r, err = kv.Diff(version, fm, tm, r)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}
