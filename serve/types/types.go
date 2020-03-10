package types

import "roci.dev/diff-server/kv"

type PullRequest struct {
	BaseStateID string `'json:"baseStateID"`
	Checksum    string `'json:"checksum"`
}

type PullResponse struct {
	StateID  string         `json:"stateID"`
	Patch    []kv.Operation `json:"patch"`
	Checksum string         `json:"checksum"`
}
