package serve

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestCheckAccess(t *testing.T) {
	assert := assert.New(t)

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
		addAccount    *Account
		accountID     string
		dbName        string
		token         string
		expectedError string
	}{
		{nil, "0", "db1", "", "No such account: '0'"},
		{&Account{ID: "0", Name: "a0"}, "1", "r0", "", "No such account: '1'"},
		{nil, "0", "db1", "", ""}, // public account
		{&Account{ID: "1", Name: "a1", Pubkey: pubPem}, "1", "db1", "", "authorization header is required"},
		{nil, "1", "db1", tokenOld, "token is expired by 9s"},
		{nil, "1", "db1", token2, "crypto/ecdsa: verification error"},
		{nil, "1", "db2", token, "token does not grant access to specified database"},
		{nil, "1", "db1", token, ""},
	}

	accounts := []Account{}
	for i, t := range tc {
		label := fmt.Sprintf("test case %d", i)
		if t.addAccount != nil {
			accounts = append(accounts, *t.addAccount)
		}
		err := checkAccess(t.accountID, t.dbName, t.token, accounts)
		if t.expectedError == "" {
			assert.NoError(nil, label)
		} else {
			assert.EqualError(err, t.expectedError, label)
		}
	}
}
