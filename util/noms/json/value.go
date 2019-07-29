package json

import (
	"bytes"

	"github.com/attic-labs/noms/go/types"
	nj "github.com/attic-labs/noms/go/util/json"

	"github.com/aboodman/replicant/util/chk"
)

type Value struct {
	types.Value
	Noms types.ValueReadWriter
}

func (v Value) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	nj.ToJSON(v.Value, buf, nj.ToOptions{
		Lists: true,
		Maps:  true,
	})
	return buf.Bytes(), nil
}

func (v *Value) UnmarshalJSON(data []byte) error {
	chk.NotNil(v.Noms, "Need to set Noms field to unmarshal from JSON")
	r, err := nj.FromJSON(bytes.NewReader(data), v.Noms, nj.FromOptions{})
	if err != nil {
		return err
	}
	v.Value = r
	return nil
}
