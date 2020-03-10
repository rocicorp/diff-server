// Package serve implements the Replicant http server. This includes all the Noms endpoints,
// plus a Replicant-specific sync endpoint that implements the server-side of the Replicant sync protocol.
package serve

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
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
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/julienschmidt/httprouter"

	"roci.dev/diff-server/db"
	servetypes "roci.dev/diff-server/serve/types"
)

// server is a single Replicant instance. The Replicant service runs many such instances.
type server struct {
	router *httprouter.Router
	db     *db.DB
	mu     sync.Mutex
}

func newServer(cs chunks.ChunkStore, urlPrefix string) (*server, error) {
	router := httprouter.New()
	noms := datas.NewDatabase(cs)
	db, err := db.New(noms)
	if err != nil {
		return nil, err
	}
	s := &server{router: router, db: db}
	s.router.POST(fmt.Sprintf("%s/handleSync", urlPrefix), func(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		body := bytes.Buffer{}
		_, err := io.Copy(&body, req.Body)
		logPayload(req, body.Bytes(), db)
		if err != nil {
			serverError(rw, err)
			return
		}
		var hsreq servetypes.PullRequest
		err = json.Unmarshal(body.Bytes(), &hsreq)
		if err != nil {
			serverError(rw, err)
			return
		}
		from, ok := hash.MaybeParse(hsreq.Basis)
		if !ok {
			clientError(rw, 400, "Invalid basis hash")
			return
		}
		patch, err := s.db.HandleSync(from)
		if err != nil {
			serverError(rw, err)
			return
		}
		hsresp := servetypes.PullResponse{
			CommitID: s.db.Head().Original.Hash().String(),
			Patch:    patch,
			Checksum: string(s.db.Head().Value.Checksum),
		}
		resp, err := json.Marshal(hsresp)
		if err != nil {
			serverError(rw, err)
			return
		}
		rw.Header().Set("Content-type", "application/json")
		rw.Header().Set("Entity-length", strconv.Itoa(len(resp)))

		w := io.Writer(rw)
		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			rw.Header().Set("Content-encoding", "gzip")
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

func logPayload(req *http.Request, body []byte, d *db.DB) {
	noms := d.Noms().(datas.Database)
	r := noms.WriteValue(types.NewBlob(noms, bytes.NewReader(body)))
	noms.Flush()
	log.Printf("x-request-id: %s, payload: %s", req.Header.Get("X-Request-Id"), r.TargetHash())
}
