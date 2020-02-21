package db

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/stretchr/testify/assert"
)

func reloadDB(assert *assert.Assertions, dir string) (db *DB) {
	sp, err := spec.ForDatabase(dir)
	assert.NoError(err)

	db, err = Load(sp)
	assert.NoError(err)

	return db
}

func TestGenesis(t *testing.T) {
	assert := assert.New(t)
	db, _ := LoadTempDB(assert)
	assert.False(db.Hash().IsEmpty())
	assert.True(db.Head().Data(db.noms).Empty())
}

func TestLoadBadSpec(t *testing.T) {
	assert := assert.New(t)

	sp, err := spec.ForDatabase("http://localhost:6666") // not running, presumably
	assert.NoError(err)
	db, err := Load(sp)
	assert.Nil(db)
	assert.EqualError(err, "Get http://localhost:6666/root/: dial tcp [::1]:6666: connect: connection refused")

	srv := httptest.NewServer(http.NotFoundHandler())
	sp, err = spec.ForDatabase(srv.URL)
	assert.NoError(err)
	db, err = Load(sp)
	assert.Nil(db)
	assert.EqualError(err, "Unexpected response: Not Found: 404 page not found")
}
