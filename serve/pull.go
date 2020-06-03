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
	"net/http"
	"strconv"
	"strings"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
	zl "github.com/rs/zerolog"

	"roci.dev/diff-server/db"
	"roci.dev/diff-server/kv"
	servetypes "roci.dev/diff-server/serve/types"
	nomsjson "roci.dev/diff-server/util/noms/json"
)

func (s *Service) pull(rw http.ResponseWriter, r *http.Request) {
	l := logger(r)
	if r.Method != "POST" {
		unsupportedMethodError(rw, r.Method, l)
		return
	}

	body := bytes.Buffer{}
	_, err := io.Copy(&body, r.Body)
	if err != nil {
		serverError(rw, fmt.Errorf("could not read body: %w", err), l)
		return
	}

	var preq servetypes.PullRequest
	err = json.Unmarshal(body.Bytes(), &preq)
	if err != nil {
		serverError(rw, fmt.Errorf("could not unmarshal body to json: %w", err), l)
		return
	}

	// TODO auth
	accountName := r.Header.Get("Authorization")
	if accountName == "" {
		clientError(rw, http.StatusBadRequest, "Missing Authorization header", l)
		return
	}
	acct, ok := lookupAccount(accountName, s.accounts)
	if !ok {
		clientError(rw, http.StatusBadRequest, fmt.Sprintf("Unknown account: %s", accountName), l)
		return
	}

	if preq.ClientID == "" {
		clientError(rw, http.StatusBadRequest, "Missing clientID", l)
		return
	}

	db, err := s.GetDB(accountName, preq.ClientID)
	if err != nil {
		serverError(rw, err, l)
		return
	}

	fromHash, ok := hash.MaybeParse(preq.BaseStateID)
	if preq.BaseStateID != "" && !ok {
		clientError(rw, http.StatusBadRequest, "Invalid baseStateID", l)
		return
	}
	fromChecksum, err := kv.ChecksumFromString(preq.Checksum)
	if err != nil {
		clientError(rw, http.StatusBadRequest, "Invalid checksum", l)
		return
	}

	clientViewURL := acct.ClientViewURL
	if s.overridClientViewURL != "" {
		clientViewURL = s.overridClientViewURL
	}
	cvReq := servetypes.ClientViewRequest{
		ClientID: preq.ClientID,
	}
	syncID := r.Header.Get("X-Replicache-SyncID")
	cvInfo := maybeGetAndStoreNewClientView(db, preq.ClientViewAuth, clientViewURL, s.clientViewGetter, cvReq, syncID, l)

	head := db.Head()
	patch, err := db.Diff(fromHash, *fromChecksum, head, l)
	if err != nil {
		serverError(rw, err, l)
		return
	}
	hsresp := servetypes.PullResponse{
		StateID:        head.NomsStruct.Hash().String(),
		LastMutationID: uint64(head.Value.LastMutationID),
		Patch:          patch,
		Checksum:       string(head.Value.Checksum),
		ClientViewInfo: cvInfo,
	}
	resp, err := json.Marshal(hsresp)
	if err != nil {
		serverError(rw, err, l)
		return
	}
	rw.Header().Set("Content-type", "application/json")
	rw.Header().Set("Entity-length", strconv.Itoa(len(resp)))

	w := io.Writer(rw)
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		rw.Header().Set("Content-encoding", "gzip")
		w = gzip.NewWriter(w)
	}

	_, err = io.Copy(w, bytes.NewReader(resp))
	if err != nil {
		serverError(rw, err, l)
	}
	w.Write([]byte{'\n'})
	if c, ok := w.(io.Closer); ok {
		c.Close()
	}
}

func maybeGetAndStoreNewClientView(db *db.DB, clientViewAuth string, url string, cvg clientViewGetter, cvReq servetypes.ClientViewRequest, syncID string, l zl.Logger) servetypes.ClientViewInfo {
	clientViewInfo := servetypes.ClientViewInfo{}
	var err error
	defer func() {
		if err != nil {
			l.Info().Msgf("got error fetching clientview: %s", err)
			clientViewInfo.ErrorMessage = err.Error()
		}
	}()

	if url == "" {
		err = errors.New("not fetching new client view: no url provided via account or --client-view")
		return clientViewInfo
	}
	cvResp, cvCode, err := cvg.Get(url, cvReq, clientViewAuth, syncID)
	clientViewInfo.HTTPStatusCode = cvCode
	if err != nil {
		return clientViewInfo
	}

	err = storeClientView(db, cvResp, l)
	return clientViewInfo
}

func storeClientView(db *db.DB, cvResp servetypes.ClientViewResponse, l zl.Logger) error {
	me := kv.NewMap(db.Noms()).Edit()
	for k, JSON := range cvResp.ClientView {
		v, err := nomsjson.FromJSON(JSON, db.Noms())
		if err != nil {
			return fmt.Errorf("error parsing clientview: %w", err)
		}
		if err := me.Set(types.String(k), v); err != nil {
			return fmt.Errorf("error setting value '%s' in clientview: %w", JSON, err)
		}
	}
	m := me.Build()
	c, err := db.MaybePutData(m, cvResp.LastMutationID)
	if err != nil {
		return fmt.Errorf("error writing new commit: %w", err)
	}
	if c.NomsStruct.IsZeroValue() {
		l.Debug().Msgf("Did not write a new commit (lastMutationID %d and checksum %s are identical to head); nop", cvResp.LastMutationID, m.Checksum())
	} else {
		basis, err := c.Basis(db.Noms())
		if err != nil {
			return err
		}
		l.Debug().Msgf("Wrote new commit %s with lastMutationID %d and checksum %s (previous commit %s had lastMutationID %d and checksum %s)", c.Ref().TargetHash(), cvResp.LastMutationID, m.Checksum(), basis.Ref().TargetHash(), uint64(basis.Value.LastMutationID), basis.Value.Checksum)
	}
	return nil
}
