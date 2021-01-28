package serve

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/account"
	"roci.dev/diff-server/serve/types"
)

func TestConcurrentAccessUsingMultipleServices(t *testing.T) {
	assert := assert.New(t)
	td, _ := ioutil.TempDir("", "")
	defer func() { assert.NoError(os.RemoveAll(td)) }()

	adb, adir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(adir)) }()

	fcvg := &fakeClientViewGet{resp: types.ClientViewResponse{}, code: 200, err: nil}
	svc1 := NewService(td, account.MaxASClientViewHosts, adb, false, fcvg, true)
	svc2 := NewService(td, account.MaxASClientViewHosts, adb, false, fcvg, true)

	res := []*httptest.ResponseRecorder{
		httptest.NewRecorder(),
		httptest.NewRecorder(),
		httptest.NewRecorder(),
	}

	reqBody := `{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid", "clientViewURL": "http://localhost:8000/client-view", "version": 3}`

	req1 := httptest.NewRequest("POST", "/pull", strings.NewReader(reqBody))
	req1.Header.Add("Authorization", "sandbox")
	req2 := httptest.NewRequest("POST", "/pull", strings.NewReader(reqBody))
	req2.Header.Add("Authorization", "sandbox")
	req3 := httptest.NewRequest("POST", "/pull", strings.NewReader(reqBody))
	req3.Header.Add("Authorization", "sandbox")
	mux1 := mux.NewRouter()
	RegisterHandlers(svc1, mux1)
	mux1.ServeHTTP(res[0], req1)
	mux2 := mux.NewRouter()
	RegisterHandlers(svc2, mux2)
	mux2.ServeHTTP(res[1], req2)
	mux1.ServeHTTP(res[2], req3)

	for i, r := range res {
		assert.Equal(http.StatusOK, r.Code, fmt.Sprintf("response %d: %s", i, string(r.Body.Bytes())))
	}
}

func TestNo301(t *testing.T) {
	assert := assert.New(t)
	td, _ := ioutil.TempDir("", "")
	defer func() { assert.NoError(os.RemoveAll(td)) }()

	adb, adir := account.LoadTempDB(assert)
	defer func() { assert.NoError(os.RemoveAll(adir)) }()

	svc := NewService(td, account.MaxASClientViewHosts, adb, false, nil, true)
	r := httptest.NewRecorder()

	mux := mux.NewRouter()
	RegisterHandlers(svc, mux)
	mux.ServeHTTP(r, httptest.NewRequest("POST", "//pull", strings.NewReader(`{"accountID": "sandbox", "baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`)))
	assert.Equal(http.StatusNotFound, r.Code)
	assert.Equal("404 page not found\n", string(r.Body.Bytes()))
}
