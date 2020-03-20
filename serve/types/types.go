package types

import (
	"encoding/json"

	"roci.dev/diff-server/kv"
)

type PullRequest struct {
	BaseStateID string `'json:"baseStateID"`
	Checksum    string `'json:"checksum"`
	ClientID    string `json:"clientID`
}

type PullResponse struct {
	StateID           string         `json:"stateID"`
	LastTransactionID string         `json:"lastTransactionID"`
	Patch             []kv.Operation `json:"patch"`
	Checksum          string         `json:"checksum"`
}

type ClientViewRequest struct {
	ClientID string `json:clientID`
}

type ClientViewResponse struct {
	ClientView        json.RawMessage `json:"clientView"`
	LastTransactionID string          `json:"lastTransactionID"`
}
