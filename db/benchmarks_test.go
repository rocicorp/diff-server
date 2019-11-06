package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
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

func BenchmarkExecHTTP(b *testing.B) {
	benchmarkExec(true, b)
}

func BenchmarkExecLocal(b *testing.B) {
	benchmarkExec(false, b)
}

func benchmarkExec(http bool, b *testing.B) {
	assert := assert.New(b)
	var db *DB
	if http {
		sp, err := spec.ForDatabase("https://replicate.to/serve/sandbox/benchmark-test")
		assert.NoError(err)
		db, err = Load(sp, "test")
		assert.NoError(err)
	} else {
		var dir string
		db, dir = LoadTempDB(assert)
		fmt.Println(dir)
	}

	db.PutBundle(types.NewBlob(db.Noms(), strings.NewReader("function put(k, v) { db.put(k, v); }")))

	for n := 0; n < b.N; n++ {
		_, err := db.Exec("put", types.NewList(db.Noms(), types.String("foo"), types.Number(n)))
		assert.NoError(err)
	}
}

func BenchmarkExecBatchHTTP1(b *testing.B) {
	benchmarkExecBatch(1, true, b)
}

func BenchmarkExecBatchHTTP10(b *testing.B) {
	benchmarkExecBatch(10, true, b)
}

func BenchmarkExecBatchHTTP100(b *testing.B) {
	benchmarkExecBatch(100, true, b)
}

func BenchmarkExecBatchHTTP1000(b *testing.B) {
	benchmarkExecBatch(1000, true, b)
}

func BenchmarkExecBatchHTTP10000(b *testing.B) {
	benchmarkExecBatch(10000, true, b)
}

func BenchmarkExecBatchLocal1(b *testing.B) {
	benchmarkExecBatch(1, false, b)
}

func BenchmarkExecBatchLocal10(b *testing.B) {
	benchmarkExecBatch(10, false, b)
}

func BenchmarkExecBatchLocal100(b *testing.B) {
	benchmarkExecBatch(100, false, b)
}

func BenchmarkExecBatchLocal1000(b *testing.B) {
	benchmarkExecBatch(1000, false, b)
}

func BenchmarkExecBatchLocal10000(b *testing.B) {
	benchmarkExecBatch(10000, false, b)
}

func benchmarkExecBatch(n int, http bool, b *testing.B) {
	assert := assert.New(b)
	var db *DB
	if http {
		sp, err := spec.ForDatabase("https://replicate.to/serve/sandbox/benchmark-test")
		assert.NoError(err)
		db, err = Load(sp, "test")
		assert.NoError(err)
	} else {
		var dir string
		db, dir = LoadTempDB(assert)
		fmt.Println(dir)
	}
	db.PutBundle(types.NewBlob(db.Noms(), strings.NewReader("function put(k, v) { db.put(k, v); }")))

	batch := make([]BatchItem, n)
	for i := 0; i < n; i++ {
		batch[i].Function = "put"
		batch[i].Args = types.NewList(db.noms, types.String("foo"), types.Number(i))
	}
	assert.Equal(n, len(batch))

	for i := 0; i < b.N; i++ {
		r, err := db.ExecBatch(batch)
		assert.NoError(err)
		for _, res := range r {
			assert.Nil(res.Result)
		}
		assert.Equal(n, len(r))
	}
}
