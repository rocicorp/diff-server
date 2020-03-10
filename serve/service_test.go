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

func TestCheckAccess(t *testing.T) {
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

		svc := NewService(td, accounts)
		res := httptest.NewRecorder()

		req := httptest.NewRequest("POST", fmt.Sprintf("/%s/%s/handlePullRequest", t.accountID, t.dbName), strings.NewReader(`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000"}`))
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

func TestConcurrentAccessUsingMultipleServices(t *testing.T) {
	// TO
	assert := assert.New(t)
	td, _ := ioutil.TempDir("", "")

	accounts := []Account{
		Account{
			ID:     "sandbox",
			Name:   "Sandbox",
			Pubkey: nil,
		},
	}

	svc1 := NewService(td, accounts)
	svc2 := NewService(td, accounts)

	res := []*httptest.ResponseRecorder{
		httptest.NewRecorder(),
		httptest.NewRecorder(),
		httptest.NewRecorder(),
	}

	svc1.ServeHTTP(res[0], httptest.NewRequest("POST", "/sandbox/foo/handlePullRequest", strings.NewReader(`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000"}`)))
	svc2.ServeHTTP(res[1], httptest.NewRequest("POST", "/sandbox/foo/handlePullRequest", strings.NewReader(`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000"}`)))
	svc1.ServeHTTP(res[2], httptest.NewRequest("POST", "/sandbox/foo/handlePullRequest", strings.NewReader(`{"baseStateID": "00000000000000000000000000000000", "checksum": "00000000"}`)))

	for i, r := range res {
		assert.Equal(http.StatusOK, r.Code, fmt.Sprintf("response %d: %s", i, string(r.Body.Bytes())))
	}
}
