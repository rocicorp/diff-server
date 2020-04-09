package kv_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/noms/memstore"
)

func TestNewMap(t *testing.T) {
	assert := assert.New(t)

	noms := memstore.New()

	// Ensure checksum matches if constructed vs built.
	constructed := kv.NewMapForTest(noms, "key1", `"1"`, "key2", `"2"`)
	me := kv.NewMap(noms).Edit()
	assert.NoError(me.Set("key1", s("1")))
	assert.NoError(me.Set("key2", s("2")))
	built := me.Build()
	assert.Equal(constructed.Checksum(), built.Checksum(), "constructed %v, built %v", constructed.DebugString(), built.DebugString())
}
