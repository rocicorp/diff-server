package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
)

func TestSync(t *testing.T) {
	assert := assert.New(t)

	client, dir := LoadTempDB(assert)
	fmt.Println("client", dir)

	b := types.NewBlob(client.noms, strings.NewReader("function log(k, v) { var val = db.get(k) || []; val.push(v); db.put(k, val); }"))
	err := client.PutBundle(b)
	assert.NoError(err)

	server, dir := LoadTempDB(assert)
	fmt.Println("server", dir)

	sp, err := spec.ForDatabase(dir)
	assert.NoError(err)
	err = client.Sync(sp)
	assert.NoError(err)
	err = server.Reload()
	assert.NoError(err)

	assert.Equal(client.head.Original.Hash().String(), server.head.Original.Hash().String())

	err = server.Exec("log", types.NewList(server.noms, types.String("foo"), types.String("foo")))
	assert.NoError(err)

	err = client.Exec("log", types.NewList(client.noms, types.String("foo"), types.String("bar")))
	assert.NoError(err)
	err = client.Exec("log", types.NewList(client.noms, types.String("foo"), types.String("baz")))
	assert.NoError(err)

	sp.GetDatabase().Rebase()
	err = client.Sync(sp)
	assert.NoError(err)

	server.Reload()

	assert.True(client.head.Original.Equals(server.head.Original))
	assert.True(types.NewList(client.noms, types.String("foo"), types.String("bar"), types.String("baz")).Equals(client.head.Data(client.noms).Get(types.String("foo"))))

	// TODO: more
}
