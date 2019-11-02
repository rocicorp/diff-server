package json

import (
	"bytes"

	"github.com/attic-labs/noms/go/types"
	nj "github.com/attic-labs/noms/go/util/json"

	"roci.dev/replicant/util/chk"
)

type Value struct {
	types.Value
	Noms types.ValueReadWriter
}

func New(noms types.ValueReadWriter, v types.Value) *Value {
	r := Make(noms, v)
	return &r
}

func Make(noms types.ValueReadWriter, v types.Value) Value {
	return Value{
		Noms:  noms,
		Value: v,
	}
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
