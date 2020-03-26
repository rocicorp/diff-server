package serve

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func DISABLED_TestCheckAccess(t *testing.T) {
	assert := assert.New(t)

	td, _ := ioutil.TempDir("", "")

	defer func() func() {
		o := jwt.TimeFunc
		jwt.TimeFunc = func() time.Time {
			return time.Unix(10, 0)
		}
		return func() {
			jwt.TimeFunc = o
		}
	}()()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(err)
	priv2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(err)

	pub, err := x509.MarshalPKIXPublicKey(priv.Public())
	assert.NoError(err)
	pubPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pub,
	})

	token, err := jwt.NewWithClaims(jwt.SigningMethodES256, Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: 11,
		},
		DB: "db1",
	}).SignedString(priv)
	assert.NoError(err)
	token2, err := jwt.NewWithClaims(jwt.SigningMethodES256, Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: 11,
		},
		DB: "db1",
	}).SignedString(priv2)
	assert.NoError(err)
	tokenOld, err := jwt.NewWithClaims(jwt.SigningMethodES256, Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: 1,
		},
		DB: "db1",
	}).SignedString(priv)
	assert.NoError(err)

	tc := []struct {
		addAccount          *Account
		accountID           string
		dbName              string
		token               string
		expectedClientError string
		expectedAuthError   string
	}{
		{nil, "0", "db1", "", "No such account: '0'", ""},
		{&Account{ID: "0", Name: "a0"}, "1", "r0", "", "No such account: '1'", ""},
		{nil, "0", "db1", "", "", ""}, // public account
		{&Account{ID: "1", Name: "a1", Pubkey: pubPem}, "1", "db1", "", "", "Authorization header is required"},
		{nil, "1", "db1", tokenOld, "", "Invalid JWT: token is expired by 9s"},
		{nil, "1", "db1", token2, "", "Invalid JWT: crypto/ecdsa: verification error"},
		{nil, "1", "db2", token, "", "Token does not grant access to specified database"},
		{nil, "1", "db1", token, "", ""},
	}

	accounts := []Account{}
	for i, t := range tc {
		label := fmt.Sprintf("test case %d", i)
		if t.addAccount != nil {
			accounts = append(accounts, *t.addAccount)
		}

		svc := NewService(td, accounts, "", nil)
		res := httptest.NewRecorder()

		req := httptest.NewRequest("POST", fmt.Sprintf("/%s/pull", t.dbName),
			strings.NewReader(fmt.Sprintf(`{"accountID": "%s", baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`, t.accountID)))
		req.Header.Add("Authorization", t.token)
		svc.ServeHTTP(res, req)

		if t.expectedClientError != "" {
			assert.Equal(http.StatusBadRequest, res.Code, label)
			assert.Equal(t.expectedClientError, string(res.Body.Bytes()))
		}
		if t.expectedAuthError != "" {
			assert.Equal(http.StatusForbidden, res.Code, label)
			assert.Equal(t.expectedAuthError, string(res.Body.Bytes()))
		}
		if t.expectedClientError == "" && t.expectedAuthError == "" {
			if res.Code != http.StatusOK {
				assert.Fail(label, "Code %d, body: %s", res.Code, string(res.Body.Bytes()))
			}
		}
	}
}

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
	// TO
	assert := assert.New(t)
	td, _ := ioutil.TempDir("", "")

	accounts := getAccounts()

	svc1 := NewService(td, accounts, "", nil)
	svc2 := NewService(td, accounts, "", nil)

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

func TestInvalidMethodPull(t *testing.T) {
	assert := assert.New(t)
	td, _ := ioutil.TempDir("", "")

	svc := NewService(td, getAccounts(), "", nil)
	r := httptest.NewRecorder()

	svc.ServeHTTP(r, httptest.NewRequest("GET", "/pull", strings.NewReader(`{"accountID": "sandbox", "baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`)))
	assert.Equal(http.StatusMethodNotAllowed, r.Code)
	assert.Equal("Unsupported method: GET", string(r.Body.Bytes()))
}

func TestNo301(t *testing.T) {
	assert := assert.New(t)
	td, _ := ioutil.TempDir("", "")

	svc := NewService(td, getAccounts(), "", nil)
	r := httptest.NewRecorder()

	svc.ServeHTTP(r, httptest.NewRequest("POST", "//pull", strings.NewReader(`{"accountID": "sandbox", "baseStateID": "00000000000000000000000000000000", "checksum": "00000000", "clientID": "clientid"}`)))
	assert.Equal(http.StatusNotFound, r.Code)
	assert.Equal("", string(r.Body.Bytes()))
}
