package db

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"
	"roci.dev/replicant/api/shared"
)

func TestRequestSync(t *testing.T) {
	assert := assert.New(t)

	remote, rdir := LoadTempDB(assert)
	local, ldir := LoadTempDB(assert)

	fmt.Println("remote", rdir)
	fmt.Println("local", ldir)

	var err error
	const code = "function append(k, v) { var list = db.get(k) || []; list.push(v); db.put(k, list); }"

	err = remote.PutBundle(types.NewBlob(remote.noms, strings.NewReader(code)))
	assert.NoError(err)

	_, err = remote.Exec("append", types.NewList(remote.noms, types.String("foo"), types.String("bar")))
	assert.NoError(err)
	_, err = remote.Exec("append", types.NewList(remote.noms, types.String("foo"), types.String("bar")))
	assert.NoError(err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req shared.HandleSyncRequest
		var err error
		err = json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(err)
		h, _ := hash.MaybeParse(req.Basis)
		patch, err := remote.HandleSync(h)
		assert.NoError(err)

		res, err := json.Marshal(shared.HandleSyncResponse{
			Patch:        patch,
			CommitID:     remote.head.Original.Hash().String(),
			NomsChecksum: remote.head.Data(remote.noms).Hash().String(),
		})
		assert.NoError(err)

		_, err = w.Write(res)
		assert.NoError(err)
	}))
	defer server.Close()

	sp, err := spec.ForDatabase(server.URL)
	assert.NoError(err)

	err = local.RequestSync(sp)
	assert.NoError(err)
}
