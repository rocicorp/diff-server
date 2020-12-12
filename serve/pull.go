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

	"roci.dev/diff-server/account"
	"roci.dev/diff-server/db"
	"roci.dev/diff-server/kv"
	servetypes "roci.dev/diff-server/serve/types"
	nomsjson "roci.dev/diff-server/util/noms/json"
)

func (s *Service) pull(rw http.ResponseWriter, r *http.Request) {
	l := logger(r)
	if r.Method != "OPTIONS" && r.Method != "POST" {
		unsupportedMethodError(rw, r.Method, l)
		return
	}
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-type, Referer, User-agent, X-Replicache-SyncID")
	if r.Method == "OPTIONS" {
		rw.WriteHeader(200)
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

	if preq.Version != 2 {
		clientError(rw, http.StatusBadRequest, "Unsupported PullRequest version", l)
		return
	}

	accountName := r.Header.Get("Authorization")
	if accountName == "" {
		clientError(rw, http.StatusBadRequest, "Missing Authorization header", l)
		return
	}
	accounts, err := account.ReadRecords(s.accountDB)
	if err != nil {
		serverError(rw, err, l)
		return
	}
	acct, ok := account.Lookup(accounts, accountName)
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

	clientViewURL := ""
	// TODO enable multiple client view URLs and auto-add ASID URLs.
	if len(acct.ClientViewURLs) > 0 {
		clientViewURL = acct.ClientViewURLs[0]
	}
	if s.overridClientViewURL != "" {
		clientViewURL = s.overridClientViewURL
	}
	cvReq := servetypes.ClientViewRequest{
		ClientID: preq.ClientID,
	}
	syncID := r.Header.Get("X-Replicache-SyncID")
	head := db.Head()
	// minLastMutationID is the smallest last mutation id we will accept from the client view
	minLastMutationID := uint64(head.Value.LastMutationID)
	if preq.LastMutationID > minLastMutationID {
		minLastMutationID = preq.LastMutationID
	}
	cvInfo := maybeGetAndStoreNewClientView(db, preq.ClientViewAuth, clientViewURL, s.clientViewGetter, cvReq, minLastMutationID, syncID, l)

	head = db.Head() // head could have changed in maybeGetAndStoreNewClientView
	var presp servetypes.PullResponse
	if uint64(head.Value.LastMutationID) < preq.LastMutationID {
		// Refuse to send the client backwards in time.
		presp = nopPull(&preq, &cvInfo)
	} else {
		patch, err := db.Diff(preq.Version, fromHash, *fromChecksum, head, l)
		if err != nil {
			serverError(rw, err, l)
			return
		}
		presp = servetypes.PullResponse{
			StateID:        head.NomsStruct.Hash().String(),
			LastMutationID: uint64(head.Value.LastMutationID),
			Patch:          patch,
			Checksum:       string(head.Value.Checksum),
			ClientViewInfo: cvInfo,
		}
	}
	resp, err := json.Marshal(presp)
	if err != nil {
		serverError(rw, err, l)
		return
	}
	// Add a newline to make output to console etc nicer.
	resp = append(resp, byte('\n'))
	rw.Header().Set("Content-type", "application/json")
	rw.Header().Set("Entity-length", strconv.Itoa(len(resp)))

	w := io.Writer(rw)
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		rw.Header().Set("Content-encoding", "gzip")
		gzw := gzip.NewWriter(rw)
		defer gzw.Close()
		w = gzw
	}
	_, err = io.Copy(w, bytes.NewReader(resp))
	if err != nil {
		serverError(rw, err, l)
		return
	}
}

func nopPull(pullReq *servetypes.PullRequest, cvInfo *servetypes.ClientViewInfo) servetypes.PullResponse {
	return servetypes.PullResponse{
		StateID:        pullReq.BaseStateID,
		LastMutationID: pullReq.LastMutationID,
		Patch:          make([]kv.Operation, 0),
		Checksum:       pullReq.Checksum,
		ClientViewInfo: *cvInfo,
	}
}

func maybeGetAndStoreNewClientView(db *db.DB, clientViewAuth string, url string, cvg clientViewGetter, cvReq servetypes.ClientViewRequest, minLastMutationID uint64, syncID string, l zl.Logger) servetypes.ClientViewInfo {
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

	// Refuse to go backwards in time. minLastMutationID is the greater of
	// the last mutation id of the client and head, the minimum lmid we will
	// accept from the client view.
	if cvResp.LastMutationID >= minLastMutationID {
		err = storeClientView(db, cvResp, l)
	}
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
