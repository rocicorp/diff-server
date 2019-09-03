// Package API implements the high-level API that is exposed to clients.
// Since we have many clients in many different languages, this is implemented
// language/host-indepedently, and further adapted by different packages.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/attic-labs/noms/go/types"

	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/chk"
	jsnoms "github.com/aboodman/replicant/util/noms/json"
)

type GetRootRequest struct {
}

type GetRootResponse struct {
	Root jsnoms.Hash `json:"root"`
}

type HasRequest struct {
	Key string `json:"key"`
}

type HasResponse struct {
	Has bool `json:"has"`
}

type GetRequest struct {
	Key string `json:"key"`
}

type GetResponse struct {
	Has  bool          `json:"has"`
	Data *jsnoms.Value `json:"data,omitempty"`
}

type ScanRequest struct {
	Prefix    string `json:"prefix,omitempty"`
	StartAtID string `json:"fromID,omitempty"`
	Limit     int    `json:"limit,omitempty"`

	// Future: EndAtID, EndBeforeID
}

type ScanItem struct {
	ID    string       `json:"id"`
	Value jsnoms.Value `json:"value"`
}

type ScanResponse struct {
	Values []ScanItem `json:"values"`
	Done   bool       `json:"done"`
}

type PutRequest struct {
	Key  string       `json:"key"`
	Data jsnoms.Value `json:"data"`
}

type PutResponse struct {
	Root jsnoms.Hash `json:"root"`
}

type DelRequest struct {
	Key string `json:"key"`
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

type SyncRequest struct {
	Remote jsnoms.Spec `json:"remote"`
}

type SyncResponse struct {
	Root jsnoms.Hash `json:"root"`
}

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
	case "sync":
		return api.dispatchSync(req)
	}
	chk.Fail("Unsupported rpc name: %s", name)
	return nil, nil
}

func (api *API) dispatchGetRoot(reqBytes []byte) ([]byte, error) {
	var req GetRootRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}

	res := GetRootResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchHas(reqBytes []byte) ([]byte, error) {
	var req HasRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	ok, err := api.db.Has(req.Key)
	if err != nil {
		return nil, err
	}
	res := HasResponse{
		Has: ok,
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchGet(reqBytes []byte) ([]byte, error) {
	var req GetRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	v, err := api.db.Get(req.Key)
	if err != nil {
		return nil, err
	}
	res := GetResponse{}
	if v == nil {
		res.Has = false
	} else {
		res.Has = true
		res.Data = jsnoms.New(api.db.Noms(), v)
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchScan(reqBytes []byte) ([]byte, error) {
	var req ScanRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	items, err := api.db.Scan(db.ScanOptions(req))
	if err != nil {
		return nil, err
	}
	res := make([]ScanItem, 0, len(items))
	for _, it := range items {
		res = append(res, ScanItem{
			ID:    it.ID,
			Value: jsnoms.Make(api.db.Noms(), it.Value),
		})
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchPut(reqBytes []byte) ([]byte, error) {
	req := PutRequest{
		Data: jsnoms.Make(api.db.Noms(), nil),
	}
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	if req.Data.Value == nil {
		return nil, errors.New("data field is required")
	}
	err = api.db.Put(req.Key, req.Data.Value)
	if err != nil {
		return nil, err
	}
	res := PutResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchDel(reqBytes []byte) ([]byte, error) {
	req := DelRequest{}
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	ok, err := api.db.Del(req.Key)
	if err != nil {
		return nil, err
	}
	res := DelResponse{
		Ok: ok,
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchGetBundle(reqBytes []byte) ([]byte, error) {
	var req GetBundleRequest
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
	res := GetBundleResponse{
		Code: sb.String(),
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchPutBundle(reqBytes []byte) ([]byte, error) {
	var req PutBundleRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}
	b := types.NewBlob(api.db.Noms(), strings.NewReader(req.Code))
	err = api.db.PutBundle(b)
	if err != nil {
		return nil, err
	}
	res := PutBundleResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchExec(reqBytes []byte) ([]byte, error) {
	req := ExecRequest{
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
	res := ExecResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	if output != nil {
		res.Result = jsnoms.New(api.db.Noms(), output)
	}
	return mustMarshal(res), nil
}

func (api *API) dispatchSync(reqBytes []byte) ([]byte, error) {
	var req SyncRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, err
	}

	err = api.db.Sync(req.Remote.Spec)
	if err != nil {
		return nil, err
	}
	res := SyncResponse{
		Root: jsnoms.Hash{
			Hash: api.db.Hash(),
		},
	}
	return mustMarshal(res), nil
}

func mustMarshal(thing interface{}) []byte {
	data, err := json.Marshal(thing)
	chk.NoError(err)
	return data
}
