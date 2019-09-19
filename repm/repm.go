// Package repm implements an Android and iOS interface to Replicant via [Gomobile](https://github.com/golang/go/wiki/Mobile).
package repm

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/attic-labs/noms/go/spec"

	"github.com/aboodman/replicant/api"
	"github.com/aboodman/replicant/db"
)

type Connection struct {
	api    *api.API
	dir    string
	origin string
	tmpDir string
}

func Open(dir, origin, tmpDir string) (*Connection, error) {
	fmt.Printf("Opening Replicant database at: %s for origin: %s\n", dir, origin)
	if tmpDir != "" {
		os.Setenv("TMPDIR", tmpDir)
	}
	fmt.Println("Using tempdir: ", os.TempDir())
	sp, err := spec.ForDatabase(dir)
	if err != nil {
		return nil, err
	}
	db, err := db.Load(sp, origin)
	if err != nil {
		return nil, err
	}
	return &Connection{api: api.New(db), dir: dir, origin: origin, tmpDir: tmpDir}, nil
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
			fmt.Fprintf(os.Stderr, "Replicant panicked with: %s\n%s\n", msg, string(debug.Stack()))
			ret = nil
			err = fmt.Errorf("Replicant panicked with: %s - see stderr for more.", msg)
		}
	}()
	switch rpc {
	case "dropDatabase":
		ret, err = nil, conn.dropDatabase()
	default:
		ret, err = conn.api.Dispatch(rpc, data)
	}
	return
}

func (conn *Connection) dropDatabase() error {
	err := os.RemoveAll(conn.dir)
	if err != nil {
		return err
	}
	newConn, err := Open(conn.dir, conn.origin, conn.tmpDir)
	if err != nil {
		return err
	}
	*conn = *newConn
	return nil
}
