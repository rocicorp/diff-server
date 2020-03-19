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

func TestClientViewGetter_Get(t *testing.T) {
	assert := assert.New(t)

	type args struct {
	}
	tests := []struct {
		name      string
		req       servetypes.ClientViewRequest
		authToken string
		respCode  int
		respBody  string
		want      servetypes.ClientViewResponse
		wantErr   string
	}{
		{
			"ok",
			servetypes.ClientViewRequest{ClientID: "clientid"},
			"authtoken",
			http.StatusOK,
			`{"clientView": "clientview", "lastTransactionID": "ltid"}`,
			servetypes.ClientViewResponse{ClientView: []byte("\"clientview\""), LastTransactionID: "ltid"},
			"",
		},
		{
			"error",
			servetypes.ClientViewRequest{ClientID: "clientid"},
			"authtoken",
			http.StatusBadRequest,
			``,
			servetypes.ClientViewResponse{},
			"400",
		},
		{
			"missing last transaction id",
			servetypes.ClientViewRequest{ClientID: "clientid"},
			"authtoken",
			http.StatusOK,
			`{"clientView": "foo", "lastTransactionID": ""}`,
			servetypes.ClientViewResponse{},
			"lastTransactionID",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var reqBody servetypes.ClientViewRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				assert.NoError(err, tt.name)
				assert.Equal(tt.req.ClientID, reqBody.ClientID, tt.name)
				assert.Equal(tt.authToken, r.Header.Get("Authorization"), tt.name)
				w.WriteHeader(tt.respCode)
				w.Write([]byte(tt.respBody))
			}))

			g := ClientViewGetter{
				url: server.URL,
			}
			got, err := g.Get(tt.req, tt.authToken)
			if tt.wantErr == "" {
				assert.NoError(err)
			} else {
				assert.Error(err)
				assert.Regexp(tt.wantErr, err.Error(), tt.name)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientViewGetter.Get() case %s got %v (clientview=%s), want %v (clientview=%s)", tt.name, got, string(got.ClientView), tt.want, string(tt.want.ClientView))
			}
		})
	}
}
