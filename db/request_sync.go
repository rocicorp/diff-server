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
)

// RequestSync kicks off the new patch-based sync protocol from the client side.
func (db *DB) RequestSync(remote spec.Spec) error {
	url := fmt.Sprintf("%s/handleSync", remote.String())
	reqBody, err := json.Marshal(shared.HandleSyncRequest{
		Basis: db.head.Meta.Genesis.ServerCommitID,
	})
	fmt.Println("Requesting basis: ", db.head.Meta.Genesis.ServerCommitID)
	chk.NoError(err)

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", remote.Options.Authorization)
	req.Header.Add("Content-Encoding", "gzip")
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
		return err
	}

	if len(respBody.Patch) == 0 {
		return nil
	}

	var patch = respBody.Patch
	head := makeGenesis(db.noms, respBody.CommitID)
	if patch[0].Op == jsonpatch.OpRemove && patch[0].Path == "/" {
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
				return err
			}
			head.Value.Code = db.noms.WriteValue(types.NewBlob(db.noms, strings.NewReader(code)))
		case strings.HasPrefix(op.Path, "/u"):
			if ed == nil {
				ed = db.head.Data(db.noms).Edit()
			}
			op.Path = op.Path[2:]
			err = jsonpatch.ApplyOne(db.noms, ed, op)
			if err != nil {
				return err
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
