package serve

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getAccounts() []Account {
	return []Account{
		Account{
			ID:     "sandbox",
			Name:   "Sandbox",
			Pubkey: nil,
		},
	}
}

func TestConcurrentAccessUsingMultipleServices(t *testing.T) {
	assert := assert.New(t)
	td, _ := ioutil.TempDir("", "")

	accounts := getAccounts()

	svc1 := NewService(td, accounts, "", nil, true)
	svc2 := NewService(td, accounts, "", nil, true)

	res := []*httptest.ResponseRecorder{
		httptest.NewRecorder(),
		httptest.NewRecorder(),
		httptest.NewRecorder(),
	}

	req1 := httptest.NewRequest("POST", "/pull", strings.NewReader(`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`))
	req1.Header.Add("Authorization", "sandbox")
	req2 := httptest.NewRequest("POST", "/pull", strings.NewReader(`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`))
	req2.Header.Add("Authorization", "sandbox")
	req3 := httptest.NewRequest("POST", "/pull", strings.NewReader(`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`))
	req3.Header.Add("Authorization", "sandbox")
	svc1.ServeHTTP(res[0], req1)
	svc2.ServeHTTP(res[1], req2)
	svc1.ServeHTTP(res[2], req3)

	for i, r := range res {
		assert.Equal(http.StatusOK, r.Code, fmt.Sprintf("response %d: %s", i, string(r.Body.Bytes())))
	}
}

func TestNo301(t *testing.T) {
	assert := assert.New(t)
	td, _ := ioutil.TempDir("", "")

	svc := NewService(td, getAccounts(), "", nil, true)
	r := httptest.NewRecorder()

	svc.ServeHTTP(r, httptest.NewRequest("POST", "//pull", strings.NewReader(`{"accountID": "sandbox", "baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`)))
	assert.Equal(http.StatusNotFound, r.Code)
	assert.Equal("not found", string(r.Body.Bytes()))
}
