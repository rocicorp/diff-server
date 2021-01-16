package types

import (
	"encoding/json"

	"roci.dev/diff-server/kv"
)

type PullRequest struct {
	// Version 0 -> uses raw json kv.Operation.Value
	// Version 1 -> uses stringified json kv.Operation.ValueString
	// Version 2 -> top-level remove uses replace path="" value="{}" instead of remove path="/"
	// Version 3 -> request explicitly specifies client view URL
	Version        uint32 `json:"version"`
	ClientViewURL  string `json:"clientViewURL"`
	ClientViewAuth string `json:"clientViewAuth"`
	ClientID       string `json:"clientID"`
	BaseStateID    string `json:"baseStateID"`
	Checksum       string `json:"checksum"`
	LastMutationID uint64 `json:"lastMutationID"`
}

type PullResponse struct {
	StateID        string         `json:"stateID"`
	LastMutationID uint64         `json:"lastMutationID"`
	Patch          []kv.Operation `json:"patch"`
	Checksum       string         `json:"checksum"`
	ClientViewInfo ClientViewInfo `json:"clientViewInfo"`
}

type ClientViewInfo struct {
	HTTPStatusCode int    `json:"httpStatusCode"`
	ErrorMessage   string `json:"errorMessage"`
}

type ClientViewRequest struct {
	ClientID string `json:"clientID"`
}

type ClientViewResponse struct {
	ClientView     map[string]json.RawMessage `json:"clientView"`
	LastMutationID uint64                     `json:"lastMutationID"`
}

type InjectRequest struct {
	AccountID          string             `json:"accountID"`
	ClientID           string             `json:"clientID"`
	ClientViewResponse ClientViewResponse `json:"clientViewResponse"`
}
