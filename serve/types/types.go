package types

import "roci.dev/diff-server/kv"

type HandleSyncRequest struct {
	Basis string `'json:"basis"`
}

type HandleSyncResponse struct {
	CommitID     string         `json:"commitID"`
	Patch        []kv.Operation `json:"patch"`
	NomsChecksum string         `json:"nomsChecksum"`
}
