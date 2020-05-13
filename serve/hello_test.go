package serve

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/util/log"
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
		s := NewService(td, []Account{Account{ID: "accountID", Name: "accountID", Pubkey: nil}}, "", nil, true)

		msg := fmt.Sprintf("test case %d", i)
		req := httptest.NewRequest(t.method, "/hello", nil)
		resp := httptest.NewRecorder()
		s.hello(resp, req, log.Default())

		body := bytes.Buffer{}
		_, err := io.Copy(&body, resp.Result().Body)
		assert.NoError(err, msg)
		assert.Equal(t.wantRespCode, resp.Result().StatusCode, msg)
		assert.Regexp(t.wantRespBody, string(body.Bytes()))
	}
}
