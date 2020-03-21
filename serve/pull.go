// Package serve implements the Replicant http server. This includes all the Noms endpoints,
// plus a Replicant-specific sync endpoint that implements the server-side of the Replicant sync protocol.
package serve

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"

	"roci.dev/diff-server/db"
	"roci.dev/diff-server/kv"
	servetypes "roci.dev/diff-server/serve/types"
	nomsjson "roci.dev/diff-server/util/noms/json"
)

func (s *Service) pull(rw http.ResponseWriter, req *http.Request) {
	body := bytes.Buffer{}
	_, err := io.Copy(&body, req.Body)

	var preq servetypes.PullRequest
	err = json.Unmarshal(body.Bytes(), &preq)
	if err != nil {
		serverError(rw, err)
		return
	}

	if preq.AccountID == "" {
		clientError(rw, http.StatusBadRequest, "Missing accountID")
		return
	}

	_, ok := lookupAccount(preq.AccountID, s.accounts)
	if !ok {
		clientError(rw, http.StatusBadRequest, "Unknown accountID")
		return
	}

	// TODO: auth

	if preq.ClientID == "" {
		clientError(rw, http.StatusBadRequest, "Missing clientID")
		return
	}

	db, err := s.GetDB(preq.AccountID, preq.ClientID)
	if err != nil {
		serverError(rw, err)
		return
	}

	logPayload(req, body.Bytes(), db.Noms())

	from, ok := hash.MaybeParse(preq.BaseStateID)
	if !ok {
		clientError(rw, http.StatusBadRequest, "Invalid baseStateID")
		return
	}
	fromChecksum, err := kv.ChecksumFromString(preq.Checksum)
	if err != nil {
		clientError(rw, http.StatusBadRequest, "Invalid checksum")
	}

	cvReq := servetypes.ClientViewRequest{ClientID: preq.ClientID}
	maybeGetAndStoreNewClientView(db, req, s.cvg, cvReq)

	patch, err := db.Diff(from, *fromChecksum)
	if err != nil {
		serverError(rw, err)
		return
	}
	hsresp := servetypes.PullResponse{
		StateID:           db.Head().Original.Hash().String(),
		LastTransactionID: string(db.Head().Value.LastTransactionID),
		Patch:             patch,
		Checksum:          string(db.Head().Value.Checksum),
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
}

func maybeGetAndStoreNewClientView(db *db.DB, pullHttpReq *http.Request, cvg clientViewGetter, cvReq servetypes.ClientViewRequest) {
	var err error
	defer func() {
		if err != nil {
			log.Printf("WARNING: got error fetching clientview: %s", err)
		}
	}()

	if cvg == nil {
		err = errors.New("not fetching new client view: no url provided via account or --clientview")
		return
	}
	cvResp, err := cvg.Get(cvReq, pullHttpReq.Header.Get("Authorization"))
	if err != nil {
		return
	}
	v, err := nomsjson.FromJSON(bytes.NewReader(cvResp.ClientView), db.Noms())
	if err != nil {
		return
	}
	nm, ok := v.(types.Map)
	if !ok {
		err = fmt.Errorf("clientview is not a json object, it looks to noms like a %s", v.Kind().String())
		return
	}
	// TODO fritz yes this is inefficient, will fix up Map so we don't have to go
	// back and forth. But after it works.
	m := kv.NewMapFromNoms(db.Noms(), nm)
	if m == nil {
		err = errors.New("couldnt create a Map from a Noms Map")
	}
	err = db.PutData(m.NomsMap(), types.String(m.Checksum().String()), cvResp.LastTransactionID)
	return
}

func logPayload(req *http.Request, body []byte, noms datas.Database) {
	r := noms.WriteValue(types.NewBlob(noms, bytes.NewReader(body)))
	noms.Flush()
	log.Printf("x-request-id: %s, payload: %s", req.Header.Get("X-Request-Id"), r.TargetHash())
}
