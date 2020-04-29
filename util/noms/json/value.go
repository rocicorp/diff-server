package json

import (
	"bytes"

	"github.com/attic-labs/noms/go/types"
	"roci.dev/diff-server/util/chk"
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
	err := ToJSON(v.Value, buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (v *Value) UnmarshalJSON(data []byte) error {
	chk.NotNil(v.Noms, "Need to set Noms field to unmarshal from JSON")
	r, err := FromJSON(bytes.NewReader(data), v.Noms)
	if err != nil {
		return err
	}
	v.Value = r
	return nil
}
