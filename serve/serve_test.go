package serve

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/db"
)

func TestBasics(t *testing.T) {
	assert := assert.New(t)

	go func() {
		serverDir, err := ioutil.TempDir("", "")
		assert.NoError(err)
		sp, err := spec.ForDatabase(serverDir)
		assert.NoError(err)
		s, err := NewServer(sp.NewChunkStore(), "")
		assert.NoError(err)
		http.Handle("/", s)
		assert.NoError(http.ListenAndServe(":8674", nil))
	}()

	d, dir := db.LoadTempDB(assert)
	fmt.Println("client test db", dir)
	assert.NoError(d.PutBundle(types.NewBlob(d.Noms(), strings.NewReader(`function push(key, val) { list = db.get(key) || []; list.push(val); db.put(key, list); }`))))
	_, err := d.Exec("push", types.NewList(d.Noms(), types.String("items"), types.String("foo")))
	assert.NoError(err)
	_, err = d.Exec("push", types.NewList(d.Noms(), types.String("items"), types.String("bar")))
	assert.NoError(err)

	sp, err := spec.ForDatabase("http://localhost:8674")
	assert.NoError(err)
	err = d.Sync(sp)
	assert.NoError(err)

	remote, err := db.New(sp.GetDatabase(), "")
	remote.Reload()
	assert.NoError(err)
	assert.Equal(d.Hash(), remote.Hash())
}
