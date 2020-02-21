package json

import (
	"fmt"

	"github.com/attic-labs/noms/go/types"

	"roci.dev/diff-server/util/chk"
)

// TODO: test
type List struct {
	Value
}

func MakeList(noms types.ValueReadWriter, v types.Value) List {
	if v != nil {
		chk.Equal(types.ListKind, v.Kind())
	}
	return List{
		Value: Make(noms, v),
	}
}

func (l *List) UnmarshalJSON(data []byte) error {
	temp := l.Value
	err := temp.UnmarshalJSON(data)
	if err != nil {
		return err
	}
	if temp.Kind() != types.ListKind {
		return fmt.Errorf("Unexpected Noms type: %s", types.TypeOf(temp.Value).Describe())
	}
	l.Value = temp
	return nil
}

func (l *List) List() types.List {
	return l.Value.Value.(types.List)
}
