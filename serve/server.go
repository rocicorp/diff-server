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
	"roci.dev/diff-server/kv"
	servetypes "roci.dev/diff-server/serve/types"
	"roci.dev/diff-server/util/chk"
	nomsjson "roci.dev/diff-server/util/noms/json"
)

type clientViewGetter interface {
	Get(req servetypes.ClientViewRequest, authToken string) (servetypes.ClientViewResponse, error)
}

// server is a single Replicant instance. The Replicant service runs many such instances.
type server struct {
	router *httprouter.Router
	db     *db.DB
	cvg    clientViewGetter
	mu     sync.Mutex
}

func newServer(cs chunks.ChunkStore, urlPrefix string, cvg clientViewGetter) (*server, error) {
	router := httprouter.New()
	noms := datas.NewDatabase(cs)
	db, err := db.New(noms)
	if err != nil {
		return nil, err
	}
	s := &server{router: router, db: db, cvg: cvg}
	s.router.POST(fmt.Sprintf("%s/handlePullRequest", urlPrefix), func(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		body := bytes.Buffer{}
		_, err := io.Copy(&body, req.Body)
		logPayload(req, body.Bytes(), db)
		if err != nil {
			serverError(rw, err)
			return
		}
		var preq servetypes.PullRequest
		err = json.Unmarshal(body.Bytes(), &preq)
		if err != nil {
			serverError(rw, err)
			return
		}
		from, ok := hash.MaybeParse(preq.BaseStateID)
		if !ok {
			clientError(rw, 400, "Invalid baseStateID")
			return
		}
		fromChecksum, err := kv.ChecksumFromString(preq.Checksum)
		if err != nil {
			clientError(rw, 400, "Invalid checksum")
		}
		if preq.ClientID == "" {
			clientError(rw, 400, "Missing ClientID")
			return
		}
		if cvg == nil {
			log.Print("WARNING: not fetching new client view: no url provided via account or --clientview")
		} else {
			var cvgError error
			cvreq := servetypes.ClientViewRequest{ClientID: preq.ClientID}
			cvresp, cvgError := cvg.Get(cvreq, "") // auth token TODO
			if cvgError == nil {
				cvgError = storeNewClientView(db, cvresp.ClientView)
			}
			if cvgError != nil {
				log.Printf("WARNING: got error fetching clientview: %s", cvgError)
			}
		}

		patch, err := s.db.Diff(from, *fromChecksum)
		if err != nil {
			serverError(rw, err)
			return
		}
		hsresp := servetypes.PullResponse{
			StateID:  s.db.Head().Original.Hash().String(),
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

func storeNewClientView(db *db.DB, clientView json.RawMessage) error {
	v, err := nomsjson.FromJSON(bytes.NewReader(clientView), db.Noms())
	if err != nil {
		return err
	}
	nm, ok := v.(types.Map)
	if !ok {
		return fmt.Errorf("clientview is not a Map, it's a %s", v.Kind().String())
	}
	// TODO fritz yes this is inefficient, will fix up Map so we don't have to go
	// back and forth. But after it works.
	m := kv.NewMapFromNoms(db.Noms(), nm)
	if m == nil {
		chk.Fail("couldnt create a Map from a Noms Map")
	}
	return db.PutData(m.NomsMap(), types.String(m.Checksum().String()))
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
