package serve

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"roci.dev/replicant/db"
	"roci.dev/replicant/util/time"
)

func TestAPI(t *testing.T) {
	assert := assert.New(t)

	defer time.SetFake()()

	db, s := startTestServer(assert)
	db.Put("foo", types.String("bar"))
	defer s.Shutdown(context.Background())

	tc := []struct {
		rpc              string
		req              string
		expectedResponse string
		expectedError    string
	}{
		{"handleSync", `{"basis": "00000000000000000000000000000000"}`,
			`{"patch":[{"op":"remove","path":"/"},{"op":"add","path":"/u/foo","value":"bar"}],"commitID":"nti2kt1b288sfhdmqkgnjrog52a7m8ob","nomsChecksum":"am8lvhrbscqkngg75jaiubirapurghv9"}`, ""},
	}

	for i, t := range tc {
		msg := fmt.Sprintf("test case %d: %s: %s", i, t.rpc, t.req)
		resp, err := http.Post(fmt.Sprintf("http://localhost:8674/%s", t.rpc), "application/json", strings.NewReader(t.req))
		assert.NoError(err, msg)
		assert.Equal("application/json", resp.Header.Get("Content-type"))
		body := bytes.Buffer{}
		_, err = io.Copy(&body, resp.Body)
		assert.NoError(err, msg)
		assert.Equal(t.expectedResponse+"\n", string(body.Bytes()), msg)
	}
}

func startTestServer(assert *assert.Assertions) (*db.DB, *http.Server) {
	svr := make(chan *http.Server)
	var d *db.DB
	go func() {
		serverDir, err := ioutil.TempDir("", "")
		fmt.Printf("server dir: %s\n", serverDir)
		assert.NoError(err)
		sp, err := spec.ForDatabase(serverDir)
		assert.NoError(err)
		s, err := newServer(sp.NewChunkStore(), "")
		assert.NoError(err)
		d = s.db
		hs := http.Server{
			Addr:    ":8674",
			Handler: s,
		}
		svr <- &hs
		err = hs.ListenAndServe()
		assert.Equal(http.ErrServerClosed, err)
	}()
	return d, <-svr
}
