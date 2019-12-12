// Package serve implements the Replicant http server. This includes all the Noms endpoints,
// plus a Replicant-specific sync endpoint that implements the server-side of the Replicant sync protocol.
package serve

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/julienschmidt/httprouter"

	"roci.dev/replicant/api"
	"roci.dev/replicant/db"
)

var (
	commands = []string{"getRoot", "has", "get", "scan", "put", "del", "getBundle", "putBundle", "exec", "execBatch", "handleSync"}
)

// server is a single Replicant instance. The Replicant service runs many such instances.
type server struct {
	router *httprouter.Router
	db     *db.DB
	mu     sync.Mutex
	api    *api.API
}

func newServer(cs chunks.ChunkStore, urlPrefix, origin string) (*server, error) {
	router := datas.Router(cs, urlPrefix)
	noms := datas.NewDatabase(cs)
	db, err := db.New(noms, origin)
	if err != nil {
		return nil, err
	}
	s := &server{router: router, db: db, api: api.New(db)}
	for _, method := range commands {
		m := method
		s.router.POST(fmt.Sprintf("%s/%s", urlPrefix, method), func(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			body := bytes.Buffer{}
			_, err := io.Copy(&body, req.Body)
			logPayload(req, body.Bytes(), db)
			if err != nil {
				serverError(rw, err)
				return
			}
			resp, err := s.api.Dispatch(m, body.Bytes())
			if err != nil {
				// TODO: this might not be a client (4xx) error
				// Need to change API to be able to indicate user vs server error
				clientError(rw, http.StatusBadRequest, err.Error()+"\n")
				return
			}

			rw.Header().Set("Content-type", "application/json")
			rw.Header().Set("Content-encoding", "gzip")
			rw.Header().Set("Entity-length", strconv.Itoa(len(resp)))

			w := io.Writer(rw)
			if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
				w = gzip.NewWriter(w)
			}

			_, err = io.Copy(w, bytes.NewReader(resp))
			if err != nil {
				serverError(rw, err)
			}
			w.Write([]byte{'\n'})
			if c, ok := w.(io.Closer); ok {
				c.Close()
			}
		})
	}
	s.router.POST(urlPrefix+"/sync", func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		s.sync(w, req)
	})
	return s, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	verbose.SetVerbose(true)
	log.Println("Handling request: ", r.URL.String())

	defer func() {
		err := recover()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("handler panicked: %+v\n", err)
			debug.PrintStack()
		}
	}()

	s.router.ServeHTTP(w, r)
}

func (s *server) sync(w http.ResponseWriter, req *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	params := req.URL.Query()
	clientHash, ok := hash.MaybeParse(params.Get("head"))
	if !ok {
		clientError(w, http.StatusBadRequest, "invalid value for head param")
		return
	}
	var clientCommit db.Commit
	clientVal := s.db.Noms().ReadValue(clientHash)
	if clientVal == nil {
		clientError(w, http.StatusBadRequest, "Specified hash not found")
		return
	}
	err := marshal.Unmarshal(clientVal, &clientCommit)
	if err != nil {
		clientError(w, http.StatusBadRequest, "Invalid client commit")
		return
	}
	mergedCommit, err := db.HandleSync(s.db, clientCommit)
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, strings.NewReader(mergedCommit.TargetHash().String()))
}

func logPayload(req *http.Request, body []byte, d *db.DB) {
	noms := d.Noms().(datas.Database)
	r := noms.WriteValue(types.NewBlob(noms, bytes.NewReader(body)))
	noms.Flush()
	log.Printf("x-request-id: %s, payload: %s", req.Header.Get("X-Request-Id"), r.TargetHash())
}
