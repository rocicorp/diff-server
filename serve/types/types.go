package types

import "roci.dev/diff-server/kv"

type PullRequest struct {
	Basis    string `'json:"basis"`
	Checksum string `'json:"checksum"`
}

type PullResponse struct {
	CommitID string         `json:"commitID"`
	Patch    []kv.Operation `json:"patch"`
	Checksum string         `json:"checksum"`
}
