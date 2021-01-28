package serve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/account"
	"roci.dev/diff-server/db"
	"roci.dev/diff-server/kv"
	servetypes "roci.dev/diff-server/serve/types"
	"roci.dev/diff-server/util/time"
)

func TestAPI(t *testing.T) {
	assert := assert.New(t)
	defer time.SetFake()()

	unittestID := fmt.Sprintf("%d", account.UnittestID)

	tc := []struct {
		pullMethod  string
		pullReq     string
		authHeader  string
		disableAuth bool
		expCVURL    string
		expCVAuth   string
		CVResponse  servetypes.ClientViewResponse
		CVCode      int
		CVErr       error
		expPullResp string
		expPullErr  string
	}{
		// Unsupported method
		{"GET",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unsupported method: GET"},

		// Supports OPTIONS for cors headers
		{"OPTIONS",
			``,
			unittestID,
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			""},

		// Missing clientViewURL
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "version": 3}`,
			unittestID,
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			"",
			"clientViewURL not provided in request"},

		// Client view URL not authorized (service is configured with 1 max, already has 1.)
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://cv2.com", "version": 3}`,
			unittestID,
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			"",
			"clientViewURL is not authorized"},

		// Successful client view fetch (auth disabled).
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://SOME-UNAUTHORIZED-DOMAIN.com", "version": 3}`,
			"not a real account id",
			true,
			"http://SOME-UNAUTHORIZED-DOMAIN.com",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"qtpo2q166al1rlboo40rsstnt1gq812u","lastMutationID":2,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/new","valueString":"\"value\""}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful client view fetch.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"http://clientview.com",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"hoc705ifecv1c858qgbqr9jghh4d9l96","lastMutationID":2,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/new","valueString":"\"value\""}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful nop client view fetch where lastMutationID does not change.
		{"POST",
			`{"baseStateID": "s3n5j759kirvvs3fqeott07a43lk41ud", "checksum": "c4e7090d", "lastMutationID": 1, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"http://clientview.com",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 1},
			200,
			nil,
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful nop client view fetch where lastMutationID does change.
		{"POST",
			`{"baseStateID": "s3n5j759kirvvs3fqeott07a43lk41ud", "checksum": "c4e7090d", "lastMutationID": 1, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"http://clientview.com",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 77},
			200,
			nil,
			`{"stateID":"pi99ftvp6nchoej3i58flsqm8enqg4vd","lastMutationID":77,"patch":[],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Client view returns LMID < diffserver's => nop
		{"POST",
			`{"baseStateID": "s3n5j759kirvvs3fqeott07a43lk41ud", "checksum": "c4e7090d", "lastMutationID": 1, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"http://clientview.com",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 0},
			200,
			nil,
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Fetch errors out.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"http://clientview.com",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			errors.New("boom"),
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/foo","valueString":"\"bar\""}],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Diffserver has LMID < client's => nop (fetch is also erroring in this one, but that's incidental)
		{"POST",
			`{"baseStateID": "12345000000000000000000000000000", "checksum": "12345678", "lastMutationID": 22, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"http://clientview.com",
			"clientauth",
			servetypes.ClientViewResponse{},
			0,
			errors.New("boom"),
			`{"stateID":"12345000000000000000000000000000","lastMutationID":22,"patch":[],"checksum":"12345678","clientViewInfo":{"httpStatusCode":0,"errorMessage":""}}`,
			""},

		// No Authorization header.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			"",
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Missing Authorization"},

		// Unsupported version
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 1}`,
			unittestID,
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unsupported PullRequest version"},

		// Unknown account passed in Authorization header.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			"BONK",
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unknown account"},

		// No clientID passed in.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Missing clientID"},

		// Invalid baseStateID.
		{"POST",
			`{"baseStateID": "beep", "checksum": "00000000", "clientID": "clientid", "lastMutationID": 0, "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Invalid baseStateID"},

		// No baseStateID is fine (first pull).
		{"POST",
			`{"baseStateID": "", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"http://clientview.com",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"hoc705ifecv1c858qgbqr9jghh4d9l96","lastMutationID":2,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/new","valueString":"\"value\""}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Invalid checksum.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "not", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Invalid checksum"},

		// Ensure it canonicalizes the client view JSON.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "clientViewURL": "http://clientview.com", "version": 3}`,
			unittestID,
			false,
			"http://clientview.com",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"\u000b"`)}, LastMutationID: 2}, // "\u000B" is canonical
			200,
			nil,
			`{"stateID":"qv7hd0v4i49utb1gjs2hiefh1vfhjegk","lastMutationID":2,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/new","valueString":"\"\\u000B\""}],"checksum":"b2dc0d6a","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},
	}

	for i, t := range tc {
		fcvg := &fakeClientViewGet{resp: t.CVResponse, code: t.CVCode, err: t.CVErr}
		td, _ := ioutil.TempDir("", "")
		defer func() { assert.NoError(os.RemoveAll(td)) }()

		adb, adir := account.LoadTempDB(assert)
		defer func() { assert.NoError(os.RemoveAll(adir)) }()
		account.AddUnittestAccount(assert, adb)
		account.AddUnittestAccountHost(assert, adb, "clientview.com")

		s := NewService(td, 1 /* max auto-signup account view URLs */, adb, t.disableAuth, fcvg, true)
		noms, err := s.getNoms(unittestID)
		assert.NoError(err)
		db, err := db.New(noms.GetDataset("client/clientid"))
		assert.NoError(err)
		m := kv.NewMapForTest(noms, "foo", `"bar"`)
		c, err := db.MaybePutData(m, 1 /*lastMutationID*/)
		assert.NoError(err)
		assert.False(c.NomsStruct.IsZeroValue())

		msg := fmt.Sprintf("test case %d: %s", i, t.pullReq)
		req := httptest.NewRequest(t.pullMethod, "/sync", strings.NewReader(t.pullReq))
		req.Header.Set("Content-type", "application/json")
		if t.authHeader != "" {
			req.Header.Set("Authorization", t.authHeader)
		}
		req.Header.Set("X-Replicache-SyncID", "syncID")
		resp := httptest.NewRecorder()
		s.pull(resp, req)

		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Result().Body)
		assert.NoError(err, msg)
		if t.expPullErr == "" {
			assert.True(len(resp.Result().Header.Get("Access-Control-Allow-Origin")) > 0)
			assert.True(len(resp.Result().Header.Get("Access-Control-Allow-Methods")) > 0)
			assert.True(len(resp.Result().Header.Get("Access-Control-Allow-Headers")) > 0)
			if t.pullMethod == "OPTIONS" {
				assert.Equal(200, resp.Result().StatusCode)
				continue
			}
			assert.Equal("application/json", resp.Result().Header.Get("Content-type"))
			assert.Equal(t.expPullResp+"\n", string(body.Bytes()), msg)
		} else {
			assert.Regexp(t.expPullErr, string(body.Bytes()), msg)
		}
		if t.expCVURL != "" {
			assert.True(fcvg.called, msg)
			assert.Equal(t.expCVURL, fcvg.gotURL, msg)
			assert.Equal(t.expCVAuth, fcvg.gotAuth, msg)
			assert.Equal("clientid", fcvg.gotClientID)
			assert.Equal("syncID", fcvg.gotSyncID)
		}
	}
}

func TestAPIV2(t *testing.T) {
	assert := assert.New(t)
	defer time.SetFake()()

	unittestID := fmt.Sprintf("%d", account.UnittestID)

	tc := []struct {
		pullMethod  string
		pullReq     string
		authHeader  string
		accountCV   string
		expCVAuth   string
		CVResponse  servetypes.ClientViewResponse
		CVCode      int
		CVErr       error
		expPullResp string
		expPullErr  string
	}{
		// Unsupported method
		{"GET",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "version": 2}`,
			unittestID,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unsupported method: GET"},
		// Supports OPTIONS for cors headers
		{"OPTIONS",
			``,
			unittestID,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			""},
		// No client view to fetch from.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "version": 2}`,
			unittestID,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/foo","valueString":"\"bar\""}],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":0,"errorMessage":""}}`,
			""},

		// Successful client view fetch.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"cv",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"hoc705ifecv1c858qgbqr9jghh4d9l96","lastMutationID":2,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/new","valueString":"\"value\""}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful nop client view fetch where lastMutationID does not change.
		{"POST",
			`{"baseStateID": "s3n5j759kirvvs3fqeott07a43lk41ud", "checksum": "c4e7090d", "lastMutationID": 1, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"cv",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 1},
			200,
			nil,
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Successful nop client view fetch where lastMutationID does change.
		{"POST",
			`{"baseStateID": "s3n5j759kirvvs3fqeott07a43lk41ud", "checksum": "c4e7090d", "lastMutationID": 1, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"cv",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 77},
			200,
			nil,
			`{"stateID":"pi99ftvp6nchoej3i58flsqm8enqg4vd","lastMutationID":77,"patch":[],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Client view returns LMID < diffserver's => nop
		{"POST",
			`{"baseStateID": "s3n5j759kirvvs3fqeott07a43lk41ud", "checksum": "c4e7090d", "lastMutationID": 1, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"cv",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 0},
			200,
			nil,
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Fetch errors out.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"cv",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			errors.New("boom"),
			`{"stateID":"s3n5j759kirvvs3fqeott07a43lk41ud","lastMutationID":1,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/foo","valueString":"\"bar\""}],"checksum":"c4e7090d","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Diffserver has LMID < client's => nop (fetch is also erroring in this one, but that's incidental)
		{"POST",
			`{"baseStateID": "12345000000000000000000000000000", "checksum": "12345678", "lastMutationID": 22, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"cv",
			"clientauth",
			servetypes.ClientViewResponse{},
			0,
			errors.New("boom"),
			`{"stateID":"12345000000000000000000000000000","lastMutationID":22,"patch":[],"checksum":"12345678","clientViewInfo":{"httpStatusCode":0,"errorMessage":""}}`,
			""},

		// No Authorization header.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			"",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Missing Authorization"},

		// Unsupported version
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 1}`,
			unittestID,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unsupported PullRequest version"},

		// Unknown account passed in Authorization header.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			"BONK",
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Unknown account"},

		// No clientID passed in.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Missing clientID"},

		// Invalid baseStateID.
		{"POST",
			`{"baseStateID": "beep", "checksum": "00000000", "clientID": "clientid", "lastMutationID": 0, "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Invalid baseStateID"},

		// No baseStateID is fine (first pull).
		{"POST",
			`{"baseStateID": "", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"cv",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"value"`)}, LastMutationID: 2},
			200,
			nil,
			`{"stateID":"hoc705ifecv1c858qgbqr9jghh4d9l96","lastMutationID":2,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/new","valueString":"\"value\""}],"checksum":"f9ef007b","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},

		// Invalid checksum.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "not", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"",
			"",
			servetypes.ClientViewResponse{},
			0,
			nil,
			``,
			"Invalid checksum"},

		// Ensure it canonicalizes the client view JSON.
		{"POST",
			`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "lastMutationID": 0, "clientID": "clientid", "clientViewAuth": "clientauth", "version": 2}`,
			unittestID,
			"cv",
			"clientauth",
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"new": b(`"\u000b"`)}, LastMutationID: 2}, // "\u000B" is canonical
			200,
			nil,
			`{"stateID":"qv7hd0v4i49utb1gjs2hiefh1vfhjegk","lastMutationID":2,"patch":[{"op":"replace","path":"","valueString":"{}"},{"op":"add","path":"/new","valueString":"\"\\u000B\""}],"checksum":"b2dc0d6a","clientViewInfo":{"httpStatusCode":200,"errorMessage":""}}`,
			""},
	}

	for i, t := range tc {
		fcvg := &fakeClientViewGet{resp: t.CVResponse, code: t.CVCode, err: t.CVErr}
		td, _ := ioutil.TempDir("", "")
		defer func() { assert.NoError(os.RemoveAll(td)) }()

		adb, adir := account.LoadTempDB(assert)
		defer func() { assert.NoError(os.RemoveAll(adir)) }()
		account.AddUnittestAccount(assert, adb)
		account.AddUnittestAccountURL(assert, adb, t.accountCV)

		s := NewService(td, account.MaxASClientViewHosts, adb, false, fcvg, true)
		noms, err := s.getNoms(unittestID)
		assert.NoError(err)
		db, err := db.New(noms.GetDataset("client/clientid"))
		assert.NoError(err)
		m := kv.NewMapForTest(noms, "foo", `"bar"`)
		c, err := db.MaybePutData(m, 1 /*lastMutationID*/)
		assert.NoError(err)
		assert.False(c.NomsStruct.IsZeroValue())

		msg := fmt.Sprintf("test case %d: %s", i, t.pullReq)
		req := httptest.NewRequest(t.pullMethod, "/sync", strings.NewReader(t.pullReq))
		req.Header.Set("Content-type", "application/json")
		if t.authHeader != "" {
			req.Header.Set("Authorization", t.authHeader)
		}
		req.Header.Set("X-Replicache-SyncID", "syncID")
		resp := httptest.NewRecorder()
		s.pull(resp, req)

		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Result().Body)
		assert.NoError(err, msg)
		if t.expPullErr == "" {
			assert.True(len(resp.Result().Header.Get("Access-Control-Allow-Origin")) > 0)
			assert.True(len(resp.Result().Header.Get("Access-Control-Allow-Methods")) > 0)
			assert.True(len(resp.Result().Header.Get("Access-Control-Allow-Headers")) > 0)
			if t.pullMethod == "OPTIONS" {
				assert.Equal(200, resp.Result().StatusCode)
				continue
			}
			assert.Equal("application/json", resp.Result().Header.Get("Content-type"))
			assert.Equal(t.expPullResp+"\n", string(body.Bytes()), msg)
		} else {
			assert.Regexp(t.expPullErr, string(body.Bytes()), msg)
		}
		expectedCVURL := t.accountCV
		if expectedCVURL != "" {
			assert.True(fcvg.called, msg)
			assert.Equal(expectedCVURL, fcvg.gotURL, msg)
			assert.Equal(t.expCVAuth, fcvg.gotAuth, msg)
			assert.Equal("clientid", fcvg.gotClientID)
			assert.Equal("syncID", fcvg.gotSyncID)
		}
	}
}

type fakeClientViewGet struct {
	resp servetypes.ClientViewResponse
	code int
	err  error

	called      bool
	gotURL      string
	gotAuth     string
	gotClientID string
	gotSyncID   string
}

func (f *fakeClientViewGet) Get(url string, req servetypes.ClientViewRequest, authToken string, syncID string) (servetypes.ClientViewResponse, int, error) {
	f.called = true
	f.gotURL = url
	f.gotAuth = authToken
	f.gotClientID = req.ClientID
	f.gotSyncID = syncID
	return f.resp, f.code, f.err
}
