// Package api implements the high-level API that is exposed to clients.
// Since we have many clients in many different languages, this is implemented
// language/host-indepedently, and further adapted by different packages.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"

	"roci.dev/replicant/api/shared"
	"roci.dev/replicant/db"
	"roci.dev/replicant/exec"
	"roci.dev/replicant/util/chk"
	jsnoms "roci.dev/replicant/util/noms/json"
)

type API struct {
	db *db.DB
}

func New(db *db.DB) *API {
	return &API{db}
}

func (api *API) Dispatch(name string, req []byte) ([]byte, error) {
	switch name {
	case "getRoot":
		return api.dispatchGetRoot(req)
	case "has":
		return api.dispatchHas(req)
	case "get":
		return api.dispatchGet(req)
	case "scan":
		return api.dispatchScan(req)
	case "put":
		return api.dispatchPut(req)
	case "del":
		return api.dispatchDel(req)
	case "getBundle":
		return api.dispatchGetBundle(req)
	case "putBundle":
		return api.dispatchPutBundle(req)
	case "exec":
		return api.dispatchExec(req)
	case "execBatch":
		return api.dispatchExecBatch(req)
	case "sync":
		return api.dispatchSync(req)
	case "handleSync":
		return api.dispatchHandleSync(req)
	}
	chk.Fail("Unsupported rpc name: %s", name)
	return nil, nil
}

func (api *API) dispatchGetRoot(reqBytes []byte) ([]byte, error) {
	var req shared.GetRootRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}

	res := shared.GetRootResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchHas(reqBytes []byte) ([]byte, error) {
	var req shared.HasRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	ok, err := api.db.Has(req.ID)
	if err != nil {
		return nil, err
	}
	res := shared.HasResponse{
		Has: ok,
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchGet(reqBytes []byte) ([]byte, error) {
	var req shared.GetRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	v, err := api.db.Get(req.ID)
	if err != nil {
		return nil, err
	}
	res := shared.GetResponse{}
	if v == nil {
		res.Has = false
	} else {
		res.Has = true
		res.Value = jsnoms.New(api.db.Noms(), v)
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchScan(reqBytes []byte) ([]byte, error) {
	var req shared.ScanRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	items, err := api.db.Scan(exec.ScanOptions(req))
	if err != nil {
		return nil, err
	}
	return mustMarshal(items), nil
}

func (api *API) dispatchPut(reqBytes []byte) ([]byte, error) {
	req := shared.PutRequest{
		Value: jsnoms.Make(api.db.Noms(), nil),
	}
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	if req.Value.Value == nil {
		return nil, errors.New("value field is required")
	}
	err = api.db.Put(req.ID, req.Value.Value)
	if err != nil {
		return nil, err
	}
	res := shared.PutResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchDel(reqBytes []byte) ([]byte, error) {
	req := shared.DelRequest{}
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	ok, err := api.db.Del(req.ID)
	if err != nil {
		return nil, err
	}
	res := shared.DelResponse{
		Ok: ok,
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchGetBundle(reqBytes []byte) ([]byte, error) {
	var req shared.GetBundleRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	b, err := api.db.Bundle()
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	_, err = io.Copy(&sb, b.Reader())
	if err != nil {
		return nil, err
	}
	res := shared.GetBundleResponse{
		Code: sb.String(),
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchPutBundle(reqBytes []byte) ([]byte, error) {
	var req shared.PutBundleRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	b := types.NewBlob(api.db.Noms(), strings.NewReader(req.Code))
	err = api.db.PutBundle(b)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	res := shared.PutBundleResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchExec(reqBytes []byte) ([]byte, error) {
	req := shared.ExecRequest{
		Args: jsnoms.List{
			Value: jsnoms.Make(api.db.Noms(), nil),
		},
	}
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	output, err := api.db.Exec(req.Name, req.Args.List())
	if err != nil {
		return nil, err
	}
	res := shared.ExecResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	if output != nil {
		res.Result = jsnoms.New(api.db.Noms(), output)
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchExecBatch(reqBytes []byte) ([]byte, error) {
	var raw []json.RawMessage
	err := json.Unmarshal(reqBytes, &raw)
	if err != nil {
		return nil, err
	}

	batch := make([]db.BatchItem, 0, len(raw))
	for _, b := range raw {
		bri := shared.BatchRequestItem{
			Args: jsnoms.MakeList(api.db.Noms(), nil),
		}
		err = json.Unmarshal([]byte(b), &bri)
		if err != nil {
			return nil, err
		}
		batch = append(batch, db.BatchItem{
			Function: bri.Name,
			Args:     bri.Args.List(),
		})
	}

	dbRes, batchError, err := api.db.ExecBatch(batch)

	if err != nil {
		return nil, err
	}

	res := shared.ExecBatchResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}

	if batchError != nil {
		res.Error = &shared.BatchError{
			Index:  batchError.Index,
			Detail: batchError.Error(),
		}
	} else {
		res.Batch = make([]shared.BatchResponseItem, 0, len(dbRes))
		for _, item := range dbRes {
			bri := shared.BatchResponseItem{}
			if item.Result != nil {
				bri.Result = jsnoms.New(api.db.Noms(), item.Result)
			}
			res.Batch = append(res.Batch, bri)
		}
	}

	return mustMarshal(res), nil
}

func (api *API) dispatchSync(reqBytes []byte) ([]byte, error) {
	var req shared.SyncRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}

	req.Remote.Options.Authorization = req.Auth

	if req.Shallow {
		err = api.db.RequestSync(req.Remote.Spec)
	} else {
		err = api.db.Sync(req.Remote.Spec)
	}
	if err != nil {
		return nil, err
	}
	res := shared.SyncResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchHandleSync(reqBytes []byte) ([]byte, error) {
	var req shared.HandleSyncRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	var h hash.Hash
	if req.Basis != "" {
		var ok bool
		h, ok = hash.MaybeParse(req.Basis)
		if !ok {
			return nil, fmt.Errorf("Invalid basis hash")
		}
	}
	r, err := api.db.HandleSync(h)
	if err != nil {
		return nil, err
	}
	res := shared.HandleSyncResponse{
		CommitID:     api.db.Head().Original.Hash().String(),
		Patch:        r,
		NomsChecksum: api.db.Head().Data(api.db.Noms()).Hash().String(),
	}
	return mustMarshal(res), nil
}

func mustMarshal(thing interface{}) []byte {
	data, err := json.Marshal(thing)
	chk.NoError(err)
	return data
}
