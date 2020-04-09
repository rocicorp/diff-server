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

	"github.com/attic-labs/noms/go/types"
	"github.com/stretchr/testify/assert"

	"roci.dev/diff-server/util/time"
)

func TestInject(t *testing.T) {
	assert := assert.New(t)
	defer time.SetFake()()

	tc := []struct {
		injectEnabled bool
		method        string
		req           string
		wantRespCode  int
		wantRespBody  string
		wantChange    bool
	}{
		// Inject not enabled
		{false, "POST", `{"accountID": "accountID", "clientID": "clientID", "clientViewResponse": {"clientView":{"foo": "bar"}, "lastTransactionID":"1"}}`, http.StatusNotFound, ``, false},

		// Invalid method
		{true, "GET", ``, http.StatusMethodNotAllowed, `Unsupported method: GET`, false},

		// Empty request
		{true, "POST", ``, http.StatusBadRequest, `Bad request payload: EOF`, false},

		// Invalid JSON request
		{true, "POST", `!!`, http.StatusBadRequest, `Bad request payload: invalid character '!' looking for beginning of value`, false},

		// Missing account ID
		{true, "POST", `{"clientID": "clientID", "clientViewResponse": {"clientView":{}, "lastTransactionID":"1"}}`, http.StatusBadRequest, `Missing accountID`, false},

		// Unknown accountID
		{true, "POST", `{"accountID": "bonk", "clientID": "clientID", "clientViewResponse": {"clientView":{}, "lastTransactionID":"1"}}`, http.StatusBadRequest, `Unknown accountID`, false},

		// OK
		{true, "POST", `{"accountID": "accountID", "clientID": "clientID", "clientViewResponse": {"clientView":{"foo": "bar"}, "lastTransactionID":"1"}}`, http.StatusOK, ``, true},
	}

	for i, t := range tc {
		td, _ := ioutil.TempDir("", "")
		s := NewService(td, []Account{Account{ID: "accountID", Name: "accountID", Pubkey: nil}}, "", nil, t.injectEnabled)

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
			m := db.Head().Data(db.Noms())
			v, got := m.Get(types.String("foo"))
			assert.True(got, msg)
			assert.True(types.String("bar").Equals(v), msg)
		}
	}
}
