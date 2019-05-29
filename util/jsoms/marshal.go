package jsoms

import (
	"bytes"

	"github.com/attic-labs/noms/go/types"
	json "github.com/attic-labs/noms/go/util/json"

	"github.com/aboodman/replicant/util/chk"
)

type Value struct {
	types.Value
	Noms types.ValueReadWriter
}

func (v Value) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	json.ToJSON(v.Value, buf, json.ToOptions{
		Lists: true,
		Maps:  true,
	})
	return buf.Bytes(), nil
}

func (v *Value) UnmarshalJSON(data []byte) error {
	chk.NotNil(v.Noms, "Need to set Noms field to unmarshal from JSON")
	r, err := json.FromJSON(bytes.NewReader(data), v.Noms, json.FromOptions{})
	if err != nil {
		return err
	}
	v.Value = r
	return nil
}
