package repm

import (
	"fmt"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/lithammer/shortuuid"
)

func initClientID(noms datas.Database) (string, error) {
	ds := noms.GetDataset("_clientConfig")
	var cc ClientConfig
	if ds.HasHead() {
		err := marshal.Unmarshal(ds.Head(), &cc)
		if err != nil {
			return "", fmt.Errorf("Could not unmarshal clientConfig: %s", err.Error())
		}
	}
	if cc.ClientID == "" {
		cc.ClientID = uuid()
		noms.CommitValue(ds, marshal.MustMarshal(noms, cc))
	}
	return cc.ClientID, nil
}

var uuid = func() string {
	return shortuuid.New()
}

// ClientConfig is client-specific configuration stored for Replicant clients. It's not synced to servers
// or other nodes.
type ClientConfig struct {
	ClientID string
	Original types.Struct `noms:",original"`
}

func fakeUUID() func() {
	orig := uuid
	uuid = func() string {
		return "test"
	}
	return func() {
		uuid = orig
	}
}
