// Package api implements the high-level API that is exposed to clients.
// Since we have many clients in many different languages, this is implemented
// language/host-indepedently, and further adapted by different packages.
package shared

import (
	"roci.dev/replicant/exec"
	jsnoms "roci.dev/replicant/util/noms/json"
	"roci.dev/replicant/util/noms/jsonpatch"
)

type GetRootRequest struct {
}

type GetRootResponse struct {
	Root jsnoms.Hash `json:"root"`
}

type HasRequest struct {
	ID string `json:"id"`
}

type HasResponse struct {
	Has bool `json:"has"`
}

type GetRequest struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Has   bool          `json:"has"`
	Value *jsnoms.Value `json:"value,omitempty"`
}

type ScanRequest exec.ScanOptions

type ScanItem struct {
	ID    string       `json:"id"`
	Value jsnoms.Value `json:"value"`
}

type ScanResponse struct {
	Values []ScanItem `json:"values"`
	Done   bool       `json:"done"`
}

type PutRequest struct {
	ID    string       `json:"id"`
	Value jsnoms.Value `json:"value"`
}

type PutResponse struct {
	Root jsnoms.Hash `json:"root"`
}

type DelRequest struct {
	ID string `json:"id"`
}

type DelResponse struct {
	Ok   bool        `json:"ok"`
	Root jsnoms.Hash `json:"root"`
}

type GetBundleRequest struct {
}

type GetBundleResponse struct {
	Code string `json:"code"`
}

type PutBundleRequest struct {
	Code string `json:"code"`
}

type PutBundleResponse struct {
	Root jsnoms.Hash `json:"root"`
}

type ExecRequest struct {
	Name string      `json:name"`
	Args jsnoms.List `json:"args"`
}

type ExecResponse struct {
	Result *jsnoms.Value `json:"result,omitempty"`
	Root   jsnoms.Hash   `json:"root"`
}

type BatchRequestItem ExecRequest

// ExecBatchRequest contains a batch of transactions to execute with the `execBatch` command.
// This is much faster than executing them one-by-one via `exec`.
//
// If any transaction function returns an error, the entire batch is halted. Results from all
// previous transactions in the batch are returned however, nothing from the batch is committed.
type ExecBatchRequest []BatchRequestItem

type BatchResponseItem struct {
	Result *jsnoms.Value `json:"result,omitempty"`
}

type BatchError struct {
	Index  int    `json:"index"`
	Detail string `json:"detail"`
}

// ExecBatchResponse is the response for ExecBatchRequest. One of Batch or Error will be present.
type ExecBatchResponse struct {
	Batch []BatchResponseItem `json:"batch,omitempty"`
	Error *BatchError         `json:"error,omitempty"`
	Root  jsnoms.Hash         `json:"root"`
}

type SyncRequest struct {
	Remote jsnoms.Spec `json:"remote"`
	Auth   string      `json:"auth,omitempty"`

	// Shallow causes only the head of the remote server to be downloaded, not all of its history.
	// Currently this is incompatible with bidirectional sync.
	Shallow bool `json:"shallow,omitempty"`
}

type SyncResponseError struct {
	BadAuth string `json:"badAuth,omitempty"`
}

type SyncResponse struct {
	Error *SyncResponseError `json:"error,omitempty"`
	Root  jsnoms.Hash        `json:"root,omitempty"`
}

type HandleSyncRequest struct {
	Basis string `'json:"basis"`
}

type HandleSyncResponse struct {
	Patch        []jsonpatch.Operation `json:"patch"`
	CommitID     string                `json:"commitID"`
	NomsChecksum string                `json:"nomsChecksum"`
}
