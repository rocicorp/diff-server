package exec

import (
	"bytes"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/db"
)

func TestBasic(t *testing.T) {
	assert := assert.New(t)

	sp, err := spec.ForDatabase("mem")
	assert.NoError(err)

	db, err := db.Load(sp)
	assert.NoError(err)

	code := `function incr(db, d) {
	var prev = db.get('v1') || 0;
	db.put('v1', prev + d);
}
`

	err = Run(db, bytes.NewBuffer([]byte(code)), "incr", types.NewList(db.Noms(), types.Number(1)));
	assert.NoError(err)

	buf := &bytes.Buffer{}
	ok, err := db.Get("v1", buf)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("1\n", string(buf.Bytes()))

	err = Run(db, bytes.NewBuffer([]byte(code)), "incr", types.NewList(db.Noms(), types.Number(42)));
	assert.NoError(err)
	buf.Reset()
	ok, err = db.Get("v1", buf)
	assert.NoError(err)
	assert.True(ok)
	assert.Equal("43\n", string(buf.Bytes()))
}
