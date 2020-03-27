package serve

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/kv"
	"roci.dev/diff-server/util/time"
)

func TestInject(t *testing.T) {
	assert := assert.New(t)
	defer time.SetFake()()

	tc := []struct {
		method       string
		req          string
		wantRespCode int
		wantRespBody string
		wantChange   bool
	}{
		// Invalid method
		{"GET", ``, http.StatusBadRequest, `Unsupported method: GET`, false},

		// Empty request
		{"POST", ``, http.StatusBadRequest, `Bad request payload: EOF`, false},

		// Invalid JSON request
		{"POST", `!!`, http.StatusBadRequest, `Bad request payload: invalid character '!' looking for beginning of value`, false},

		// Missing account ID
		{"POST", `{"clientID": "clientID", "clientViewResponse": {"clientView":{}, "lastTransactionID":"1"}}`, http.StatusBadRequest, `Missing accountID`, false},

		// Unknown accountID
		{"POST", `{"accountID": "bonk", "clientID": "clientID", "clientViewResponse": {"clientView":{}, "lastTransactionID":"1"}}`, http.StatusBadRequest, `Unknown accountID`, false},

		// OK
		{"POST", `{"accountID": "accountID", "clientID": "clientID", "clientViewResponse": {"clientView":{"foo": "bar"}, "lastTransactionID":"1"}}`, http.StatusOK, ``, true},
	}

	for i, t := range tc {
		td, _ := ioutil.TempDir("", "")
		s := NewService(td, []Account{Account{ID: "accountID", Name: "accountID", Pubkey: nil}}, "", nil)

		msg := fmt.Sprintf("test case %d", i)
		req := httptest.NewRequest(t.method, "/inject", strings.NewReader(t.req))
		req.Header.Set("Content-type", "application/json")
		resp := httptest.NewRecorder()
		s.inject(resp, req)

		body := bytes.Buffer{}
		_, err := io.Copy(&body, resp.Result().Body)
		assert.NoError(err, msg)
		assert.Equal(t.wantRespCode, resp.Result().StatusCode, msg)
		assert.Equal(t.wantRespBody, string(body.Bytes()), msg)

		if t.wantChange {
			db, err := s.GetDB("accountID", "clientID")
			assert.NoError(err, msg)
			m := kv.NewMapFromNoms(db.Noms(), db.Head().Data(db.Noms()))
			v, err := m.Get("foo")
			assert.NoError(err, msg)
			assert.Equal("\"bar\"\n", string(v), msg)
		}
	}
}
