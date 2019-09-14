package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"github.com/aboodman/replicant/util/noms/diff"
	"github.com/aboodman/replicant/util/time"
)

func TestSyncMerge(t *testing.T) {
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

	_, err = server.Exec("log", types.NewList(server.noms, types.String("foo"), types.String("foo")))
	assert.NoError(err)

	_, err = client.Exec("log", types.NewList(client.noms, types.String("foo"), types.String("bar")))
	assert.NoError(err)
	_, err = client.Exec("log", types.NewList(client.noms, types.String("foo"), types.String("baz")))
	assert.NoError(err)

	sp.GetDatabase().Rebase()
	err = client.Sync(sp)
	assert.NoError(err)

	server.Reload()

	assert.True(client.head.Original.Equals(server.head.Original))
	assert.True(types.NewList(client.noms, types.String("foo"), types.String("bar"), types.String("baz")).Equals(client.head.Data(client.noms).Get(types.String("foo"))))
}

func TestSyncFastForward(t *testing.T) {
	defer time.SetFake()()
	assert := assert.New(t)

	client, dir := LoadTempDB(assert)
	fmt.Println("client", dir)

	b := types.NewBlob(client.noms,
		strings.NewReader(`function write(v) { db.put("foo", v); } function read() { return db.get("foo"); }`))
	err := client.PutBundle(b)
	assert.NoError(err)

	_, err = client.Exec("write", types.NewList(client.Noms(), types.String("bar")))
	assert.NoError(err)

	server, dir := LoadTempDB(assert)
	fmt.Println("server", dir)

	sp, err := spec.ForDatabase(dir)
	assert.NoError(err)
	err = client.Sync(sp)
	assert.NoError(err)
	err = server.Reload()
	assert.NoError(err)

	noms := client.Noms()
	c0 := makeGenesis(noms)
	c1 := makeTx(noms,
		c0.Ref(),
		client.origin,
		time.DateTime(),
		types.Ref{},
		".putBundle",
		types.NewList(noms, b),
		types.NewRef(b),
		c0.Value.Data)
	c2 := makeTx(noms,
		c1.Ref(),
		client.origin,
		time.DateTime(),
		types.NewRef(b),
		"write",
		types.NewList(noms, types.String("bar")),
		types.NewRef(b),
		types.NewRef(c1.Data(noms).Edit().Set(types.String("foo"), types.String("bar")).Map()))

	assert.Equal(c2.Original.Hash(), client.head.Original.Hash(), diff.Diff(c2.Original, client.head.Original))
	assert.Equal(c2.Original.Hash(), server.head.Original.Hash(), diff.Diff(c2.Original, server.head.Original))
}

// TODO: add a test for syncing a commit that is rejected and continuing
