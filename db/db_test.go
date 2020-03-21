package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenesis(t *testing.T) {
	assert := assert.New(t)
	db, _ := LoadTempDB(assert)
	assert.False(db.Hash().IsEmpty())
	assert.True(db.Head().Data(db.Noms()).Empty())
}

// hmmm.. we seem to have removed every test.
