package serve

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/db"
	"roci.dev/diff-server/kv"
	servetypes "roci.dev/diff-server/serve/types"
	"roci.dev/diff-server/util/time"
)

func TestAPI(t *testing.T) {
	assert := assert.New(t)
	defer time.SetFake()()

	tc := []struct {
		pullMethod  string
		pullReq     string
		authHeader  string
		expCVReq    *servetypes.ClientViewRequest
		expCVAuth   string
		CVResponse  servetypes.ClientViewResponse
		CVCode      int
		CVErr       error
		expPullResp string
		expPullErr  string
	}{
		// Unsupported method
		{"GET",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`,
			"accountID",
			nil,
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unsupported method: GET"},

		// No client view to fetch from.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`,
			"accountID",
			nil,
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			`{"stateID":"o9ic5cumvag1ksqln6a4jf62qdip9m8p","lastMutationID":1,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/foo","value":"bar"}],"checksum":"7d4a87ba","clientViewInfo":{"httpStatusCode":0,"errorMessage":""}}`,
			""},

		// Successful client view fetch.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			&servetypes.ClientViewRequest{},
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]interface{}{"new": "value"}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"so63u0ngdmhknauno8o06nesijj74c4v","lastMutationID":2,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/new","value":"value"}],"checksum":"2a408ef6","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful nop client view fetch where lastMutationID does not change.
		{"POST",
			`{"baseStateID": "o9ic5cumvag1ksqln6a4jf62qdip9m8p", "checksum": "7d4a87ba", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			&servetypes.ClientViewRequest{},
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]interface{}{"foo": "bar"}, LastMutationID: 1},
			200,
			nil,
			`{"stateID":"o9ic5cumvag1ksqln6a4jf62qdip9m8p","lastMutationID":1,"patch":[],"checksum":"7d4a87ba","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful nop client view fetch where lastMutationID does change.
		{"POST",
			`{"baseStateID": "o9ic5cumvag1ksqln6a4jf62qdip9m8p", "checksum": "7d4a87ba", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			&servetypes.ClientViewRequest{},
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]interface{}{"foo": "bar"}, LastMutationID: 77},
			200,
			nil,
			`{"stateID":"3mrtvk68v6otl194pnqjrcehkir19mav","lastMutationID":77,"patch":[],"checksum":"7d4a87ba","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Fetch errors out.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			&servetypes.ClientViewRequest{},
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]interface{}{"new": "value"}, LastMutationID: 2},
			200,
			errors.New("boom"),
			`{"stateID":"o9ic5cumvag1ksqln6a4jf62qdip9m8p","lastMutationID":1,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/foo","value":"bar"}],"checksum":"7d4a87ba","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// No Authorization header.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"",
			nil,
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Missing Authorization"},

		// Unknown account passed in Authorization header.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"BONK",
			nil,
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unknown account"},

		// No clientID passed in.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientViewAuth": "clientauth"}`,
			"accountID",
			nil,
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Missing clientID"},

		// Invalid baseStateID.
		{"POST",
			`{"baseStateID": "beep", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			nil,
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Invalid baseStateID"},

		// No baseStateID is fine (first pull).
		{"POST",
			`{"baseStateID": "", "checksum": "00000000", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			&servetypes.ClientViewRequest{},
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]interface{}{"new": "value"}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"so63u0ngdmhknauno8o06nesijj74c4v","lastMutationID":2,"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/new","value":"value"}],"checksum":"2a408ef6","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Invalid checksum.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "not", "clientID": "clientid", "clientViewAuth": "clientauth"}`,
			"accountID",
			nil,
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Invalid checksum"},
	}

	for i, t := range tc {
		fcvg := fakeClientViewGet{resp: t.CVResponse, code: t.CVCode, err: t.CVErr}
		var cvg clientViewGetter
		if t.expCVReq != nil {
			cvg = &fcvg
		}

		td, _ := ioutil.TempDir("", "")
		s := NewService(td, []Account{Account{ID: "accountID", Name: "accountID", Pubkey: nil}}, "", cvg, true)
		noms, err := s.getNoms("accountID")
		assert.NoError(err)
		db, err := db.New(noms.GetDataset("client/clientid"))
		assert.NoError(err)
		m := kv.WrapMapAndComputeChecksum(noms, types.NewMap(noms, types.String("foo"), types.String("bar")))
		err = db.PutData(m, 1 /*lastMutationID*/)
		assert.NoError(err)

		msg := fmt.Sprintf("test case %d: %s", i, t.pullReq)
		req := httptest.NewRequest(t.pullMethod, "/sync", strings.NewReader(t.pullReq))
		req.Header.Set("Content-type", "application/json")
		if t.authHeader != "" {
			req.Header.Set("Authorization", t.authHeader)
		}
		resp := httptest.NewRecorder()
		s.pull(resp, req)

		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Result().Body)
		assert.NoError(err, msg)
		if t.expPullErr == "" {
			assert.Equal("application/json", resp.Result().Header.Get("Content-type"))
			assert.Equal(t.expPullResp+"\n", string(body.Bytes()), msg)
		} else {
			assert.Regexp(t.expPullErr, string(body.Bytes()), msg)
		}
		if t.expCVReq != nil {
			assert.True(fcvg.called)
			assert.Equal(*t.expCVReq, fcvg.gotReq)
			assert.Equal(t.expCVAuth, fcvg.gotAuth)
		}
	}
}

type fakeClientViewGet struct {
	resp servetypes.ClientViewResponse
	code int
	err  error

	called  bool
	gotReq  servetypes.ClientViewRequest
	gotAuth string
}

func (f *fakeClientViewGet) Get(url string, req servetypes.ClientViewRequest, authToken string) (servetypes.ClientViewResponse, int, error) {
	f.called = true
	f.gotReq = req
	f.gotAuth = authToken
	return f.resp, f.code, f.err
}
