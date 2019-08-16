// package remp implements a Gomobile-compatible API to replicant.
package repm

import (
	"fmt"

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

func (conn *Connection) Dispatch(rpc string, data []byte) (ret []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			var msg string
			if e, ok := r.(error); ok {
				msg = e.Error()
			} else {
				msg = fmt.Sprintf("%v", r)
			}
			ret = nil
			err = fmt.Errorf("Replicant panicked with: %s", msg)
		}
	}()
	ret, err = conn.api.Dispatch(rpc, data)
	return
}
