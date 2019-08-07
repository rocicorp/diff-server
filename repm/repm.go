// package remp implements a Gomobile-compatible API to replicant.
package repm

import (
	"github.com/attic-labs/noms/go/spec"

	"github.com/aboodman/replicant/api"
	"github.com/aboodman/replicant/db"
)

type Connection struct {
	api *api.API
}

func Open(dbSpec, origin string) (*Connection, error) {
	sp, err := spec.ForDatabase(dbSpec)
	if err != nil {
		return nil, err
	}
	db, err := db.Load(sp, origin)
	if err != nil {
		return nil, err
	}

	return &Connection{api: api.New(db)}, nil
}

func (conn *Connection) Dispatch(rpc string, data []byte) ([]byte, error) {
	return conn.api.Dispatch(rpc, data)
}
