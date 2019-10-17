package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func BenchmarkPut(b *testing.B) {
	assert := assert.New(b)
	db, dir := LoadTempDB(assert)
	fmt.Println(dir)
	for n := 0; n < b.N; n++ {
		err := db.Put("foo", types.Number(n))
		assert.NoError(err)
	}
}

func BenchmarkExec(b *testing.B) {
	assert := assert.New(b)
	db, dir := LoadTempDB(assert)
	fmt.Println(dir)

	db.PutBundle(types.NewBlob(db.Noms(), strings.NewReader("function put(k, v) { db.put(k, v); }")))

	for n := 0; n < b.N; n++ {
		_, err := db.Exec("put", types.NewList(db.Noms(), types.String("foo"), types.Number(n)))
		assert.NoError(err)
	}
}
