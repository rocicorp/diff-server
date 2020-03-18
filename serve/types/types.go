package types

import (
	"encoding/json"

	"roci.dev/diff-server/kv"
)

type PullRequest struct {
	BaseStateID string `'json:"baseStateID"`
	Checksum    string `'json:"checksum"`
}

type PullResponse struct {
	StateID  string         `json:"stateID"`
	Patch    []kv.Operation `json:"patch"`
	Checksum string         `json:"checksum"`
}

type ClientViewRequest struct {
	ClientID string `json:clientID`
}

type ClientViewResponse struct {
	ClientView        json.RawMessage `json:clientView`
	StateID           string          `json:stateID`
	LastTransactionID string          `json:lastTransactionID`
}
