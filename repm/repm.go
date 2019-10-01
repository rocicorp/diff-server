// Package repm implements an Android and iOS interface to Replicant via [Gomobile](https://github.com/golang/go/wiki/Mobile).
package repm

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"

	"github.com/attic-labs/noms/go/spec"

	"github.com/aboodman/replicant/api"
	"github.com/aboodman/replicant/db"
	rlog "github.com/aboodman/replicant/util/log"
)

// Connection is a single open connection to Replicant.
type Connection struct {
	api *api.API
	dir string
}

// Open a replicant database. The replicantRootDir is a directory that contains
// zero or more named databases. Origin is the origin to use for all write
// transactions executed by this connection. tmpDir can be the empty string, in
// which case the OS default temp directory will be used.
//
// If the named database doesn't exist it is created. If the specified root
// directory doesn't exist, it is created.
func Open(replicantRootDir, dbName, origin, tmpDir string) (*Connection, error) {
	rlog.Init(os.Stderr, rlog.Options{Prefix: true})

	if replicantRootDir == "" {
		return nil, errors.New("replicantRootDir must be non-empty")
	}

	if dbName == "" {
		return nil, errors.New("dbName must be non-empty")
	}

	if origin == "" {
		return nil, errors.New("origin must be non-empty")
	}

	dbPath := path.Join(replicantRootDir, base64.RawURLEncoding.EncodeToString([]byte(dbName)))
	log.Printf("Opening Replicant database '%s' at '%s' for origin '%s'", dbName, dbPath, origin)
	if tmpDir != "" {
		os.Setenv("TMPDIR", tmpDir)
	}
	log.Println("Using tempdir: ", os.TempDir())
	sp, err := spec.ForDatabase(dbPath)
	if err != nil {
		return nil, err
	}
	db, err := db.Load(sp, origin)
	if err != nil {
		return nil, err
	}
	return &Connection{api: api.New(db), dir: dbPath}, nil
}

// Dispatch send an API request to Replicant, JSON-serialized parameters, and returns the response.
// For the list of supported API requests and their parameters, see the api package.
func (conn *Connection) Dispatch(rpc string, data []byte) (ret []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			var msg string
			if e, ok := r.(error); ok {
				msg = e.Error()
			} else {
				msg = fmt.Sprintf("%v", r)
			}
			log.Printf("Replicant panicked with: %s\n%s\n", msg, string(debug.Stack()))
			ret = nil
			err = fmt.Errorf("Replicant panicked with: %s - see stderr for more", msg)
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
	*conn = Connection{}
	return nil
}
