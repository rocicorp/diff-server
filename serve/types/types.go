package types

import (
	"roci.dev/diff-server/kv"
)

type PullRequest struct {
	AccountID   string `json:"accountID"`
	ClientID    string `json:"clientID`
	BaseStateID string `'json:"baseStateID"`
	Checksum    string `'json:"checksum"`
}

type PullResponse struct {
	StateID        string         `json:"stateID"`
	LastMutationID string         `json:"lastMutationID"`
	Patch          []kv.Operation `json:"patch"`
	Checksum       string         `json:"checksum"`
}

type ClientViewRequest struct {
	ClientID string `json:clientID`
}

type ClientViewResponse struct {
	ClientView     map[string]interface{} `json:"clientView"`
	LastMutationID string                 `json:"lastMutationID"`
}

type InjectRequest struct {
	AccountID          string             `json:"accountID"`
	ClientID           string             `json:"clientID"`
	ClientViewResponse ClientViewResponse `json:"clientViewResponse"`
}
