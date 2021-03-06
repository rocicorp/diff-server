package serve

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/account"
	"roci.dev/diff-server/util/time"
)

func TestHello(t *testing.T) {
	assert := assert.New(t)
	defer time.SetFake()()

	tc := []struct {
		method       string
		wantRespCode int
		wantRespBody string
	}{
		// Invalid method
		{"POST", http.StatusMethodNotAllowed, `Unsupported method: POST`},

		// OK
		{"GET", http.StatusOK, `Hello from Replicache`},
	}

	for i, t := range tc {
		td, _ := ioutil.TempDir("", "")
		defer func() { assert.NoError(os.RemoveAll(td)) }()

		adb, adir := account.LoadTempDB(assert)
		defer func() { assert.NoError(os.RemoveAll(adir)) }()

		s := NewService(td, account.MaxASClientViewHosts, adb, false, nil, true)

		msg := fmt.Sprintf("test case %d", i)
		req := httptest.NewRequest(t.method, "/hello", nil)
		resp := httptest.NewRecorder()
		s.hello(resp, req)

		body := bytes.Buffer{}
		_, err := io.Copy(&body, resp.Result().Body)
		assert.NoError(err, msg)
		assert.Equal(t.wantRespCode, resp.Result().StatusCode, msg)
		assert.Regexp(t.wantRespBody, string(body.Bytes()))
	}
}
