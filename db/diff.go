package db

import (
	"errors"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
	zl "github.com/rs/zerolog"

	"roci.dev/diff-server/kv"
)

func fullSync(db *DB, from hash.Hash, l zl.Logger) ([]kv.Operation, Commit) {
	l.Info().Msgf("Requested sync %s basis could not be found - sending a full sync", from.String())
	r := []kv.Operation{
		kv.Operation{
			Op:   kv.OpRemove,
			Path: "/",
		},
	}
	m := kv.NewMap(db.Noms())
	return r, makeCommit(db.Noms(), types.Ref{}, datetime.Epoch, db.ds.Database().WriteValue(m.NomsMap()), m.NomsChecksum(), 0 /*lastMutationID*/)
}

func (db *DB) Diff(fromHash hash.Hash, fromChecksum kv.Checksum, to Commit, l zl.Logger) ([]kv.Operation, error) {
	r := []kv.Operation{}
	v := db.Noms().ReadValue(fromHash)
	var fc Commit
	var err error
	if v == nil {
		r, fc = fullSync(db, fromHash, l)
	} else {
		err = marshal.Unmarshal(v, &fc)
		if err != nil {
			l.Error().Msgf("Requested sync basis %s is not a commit: %#v", fromHash, v)
			return nil, errors.New("Invalid baseStateID")
		}
	}

	fcChecksum, err := kv.ChecksumFromString(string(fc.Value.Checksum))
	if err != nil {
		l.Error().Msgf("Couldn't parse checksum from commit: %s", string(fc.Value.Checksum))
		return nil, errors.New("unable to parse commit checksum from db")
	} else if !fcChecksum.Equal(fromChecksum) {
		l.Error().Msgf("Checksum mismatch; %s from client, %s in db", fromChecksum.String(), fcChecksum.String())
		r, fc = fullSync(db, fromHash, l)
	}

	if !fc.Value.Data.Equals(to.Value.Data) {
		fm := fc.Data(db.Noms())
		tm := to.Data(db.Noms())
		r, err = kv.Diff(fm, tm, r)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}
