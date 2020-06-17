package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	servetypes "roci.dev/diff-server/serve/types"
)

func b(s string) []byte {
	return []byte(s)
}
func TestClientViewGetter_Get(t *testing.T) {
	assert := assert.New(t)

	type args struct {
	}
	tests := []struct {
		name           string
		req            servetypes.ClientViewRequest
		clientViewAuth string
		respCode       int
		respBody       string
		want           servetypes.ClientViewResponse
		wantCode       int
		wantErr        string
	}{
		{
			"ok",
			servetypes.ClientViewRequest{},
			"authtoken",
			http.StatusOK,
			`{"clientView": {"key": "value"}, "lastMutationID": 2}`,
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"key": b(`"value"`)}, LastMutationID: 2},
			http.StatusOK,
			"",
		},
		{
			"error",
			servetypes.ClientViewRequest{},
			"authtoken",
			http.StatusBadRequest,
			``,
			servetypes.ClientViewResponse{},
			http.StatusBadRequest,
			"400",
		},
		{
			"missing last mutation id",
			servetypes.ClientViewRequest{},
			"authtoken",
			http.StatusOK,
			`{"clientView": {"foo": "bar"}}`,
			servetypes.ClientViewResponse{ClientView: map[string]json.RawMessage{"foo": b(`"bar"`)}, LastMutationID: 0},
			http.StatusOK,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var reqBody servetypes.ClientViewRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				assert.NoError(err, tt.name)
				assert.Equal("application/json", r.Header.Get("Content-type"), tt.name)
				assert.Equal(tt.clientViewAuth, r.Header.Get("Authorization"), tt.name)
				assert.Equal("syncID", r.Header.Get("X-Replicache-SyncID"), tt.name)
				w.WriteHeader(tt.respCode)
				w.Write([]byte(tt.respBody))
			}))

			g := ClientViewGetter{}
			got, gotCode, err := g.Get(server.URL, tt.req, tt.clientViewAuth, "syncID")
			assert.Equal(tt.wantCode, gotCode)
			if tt.wantErr == "" {
				assert.NoError(err)
			} else {
				assert.Error(err)
				assert.Regexp(tt.wantErr, err.Error(), tt.name)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientViewGetter.Get() case %s got %v (clientview=%v), want %v (clientview=%v)", tt.name, got, got.ClientView, tt.want, tt.want.ClientView)
			}
		})
	}
}
