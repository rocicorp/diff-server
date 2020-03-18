package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	servetypes "roci.dev/diff-server/serve/types"
)

type ClientViewGetter struct {
	url string
}

// Get fetches a client view. It returns an error if the last transaction id or state id is missing.
func (g ClientViewGetter) Get(req servetypes.ClientViewRequest, authToken string) (servetypes.ClientViewResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return servetypes.ClientViewResponse{}, errors.Wrap(err, "could not marshal ClientViewRequest")
	}
	httpReq, err := http.NewRequest("POST", g.url, bytes.NewReader(reqBody))
	if err != nil {
		return servetypes.ClientViewResponse{}, errors.Wrap(err, "could not create client view http request")
	}
	httpReq.Header.Add("Authorization", authToken)
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return servetypes.ClientViewResponse{}, errors.Wrap(err, "error sending client view http request")
	}
	if httpResp.StatusCode != http.StatusOK {
		return servetypes.ClientViewResponse{}, fmt.Errorf("client view fetch http request returned %s", httpResp.Status)
	}
	var resp servetypes.ClientViewResponse
	var r io.Reader = httpResp.Body
	defer httpResp.Body.Close()
	err = json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return servetypes.ClientViewResponse{}, errors.Wrap(err, "couldnt decode client view response")
	}
	if resp.StateID == "" {
		return servetypes.ClientViewResponse{}, fmt.Errorf("malformed response %v missing stateID", resp)
	}
	if resp.LastTransactionID == "" {
		return servetypes.ClientViewResponse{}, fmt.Errorf("malformed response %v missing lastTransactionID", resp)
	}
	return resp, nil
}
