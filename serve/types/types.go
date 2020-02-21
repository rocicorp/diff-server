package types

import (
	"roci.dev/replicant/util/noms/jsonpatch"
)

type HandleSyncRequest struct {
	Basis string `'json:"basis"`
}

type HandleSyncResponse struct {
	Patch        []jsonpatch.Operation `json:"patch"`
	NomsChecksum string                `json:"nomsChecksum"`
}
