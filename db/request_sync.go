package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"roci.dev/replicant/api/shared"
	"roci.dev/replicant/util/chk"
	"roci.dev/replicant/util/noms/jsonpatch"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/verbose"
)

// RequestSync kicks off the new patch-based sync protocol from the client side.
func (db *DB) RequestSync(remote spec.Spec) error {
	url := fmt.Sprintf("%s/handleSync", remote.String())
	reqBody, err := json.Marshal(shared.HandleSyncRequest{
		Basis: db.head.Meta.Genesis.ServerCommitID,
	})
	verbose.Log("Syncing: %s from basis %s", url, db.head.Meta.Genesis.ServerCommitID)
	chk.NoError(err)

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", remote.Options.Authorization)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		var s string
		if err == nil {
			s = string(body)
		} else {
			s = err.Error()
		}
		return fmt.Errorf("%s: %s", resp.Status, s)
	}

	var respBody shared.HandleSyncResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return fmt.Errorf("Response from %s is not valid JSON: %s", url, err.Error())
	}

	var patch = respBody.Patch
	head := makeGenesis(db.noms, respBody.CommitID)
	if len(patch) > 0 && patch[0].Op == jsonpatch.OpRemove && patch[0].Path == "/" {
		patch = patch[1:]
	} else {
		head.Value = db.head.Value
	}

	var ed *types.MapEditor
	for _, op := range patch {
		switch {
		case op.Path == "/s/code":
			var code string
			err = json.Unmarshal([]byte(op.Value), &code)
			if err != nil {
				return fmt.Errorf("Cannot unmarshal /s/code: %s", err.Error())
			}
			head.Value.Code = db.noms.WriteValue(types.NewBlob(db.noms, strings.NewReader(code)))
		case strings.HasPrefix(op.Path, "/u"):
			if ed == nil {
				ed = db.head.Data(db.noms).Edit()
			}
			origPath := op.Path
			op.Path = op.Path[2:]
			err = jsonpatch.ApplyOne(db.noms, ed, op)
			if err != nil {
				return fmt.Errorf("Cannot unmarshal %s: %s", origPath, err.Error())
			}
		default:
			return fmt.Errorf("Unsupported JSON Patch operation: %s with path: %s", op.Op, op.Path)
		}
	}

	if ed != nil {
		head.Value.Data = db.noms.WriteValue(ed.Map())
	}
	if head.Value.Data.TargetHash().String() != respBody.NomsChecksum {
		return fmt.Errorf("Checksum mismatch! Expected %s, got %s", respBody.NomsChecksum, head.Value.Data.TargetHash())
	}
	db.noms.SetHead(db.noms.GetDataset(LOCAL_DATASET), db.noms.WriteValue(marshal.MustMarshal(db.noms, head)))
	return db.init()
}
