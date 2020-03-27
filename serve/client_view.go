package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	servetypes "roci.dev/diff-server/serve/types"
)

type ClientViewGetter struct{}

// Get fetches a client view. It returns an error if the response from the data layer doesn't have
// a lastMutationID.
func (g ClientViewGetter) Get(url string, req servetypes.ClientViewRequest, authToken string) (servetypes.ClientViewResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return servetypes.ClientViewResponse{}, fmt.Errorf("could not marshal ClientViewRequest: %w", err)
	}
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return servetypes.ClientViewResponse{}, fmt.Errorf("could not create client view http request: %w", err)
	}
	httpReq.Header.Add("Authorization", authToken)
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return servetypes.ClientViewResponse{}, fmt.Errorf("error sending client view http request: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return servetypes.ClientViewResponse{}, fmt.Errorf("client view fetch http request returned %s", httpResp.Status)
	}
	var resp servetypes.ClientViewResponse
	var r io.Reader = httpResp.Body
	defer httpResp.Body.Close()
	err = json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return servetypes.ClientViewResponse{}, fmt.Errorf("couldnt decode client view response: %w", err)
	}
	if resp.LastMutationID == 0 {
		return servetypes.ClientViewResponse{}, fmt.Errorf("malformed response %v: lastMutationID must be greater than zero", resp)
	}
	return resp, nil
}
