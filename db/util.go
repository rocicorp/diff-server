package db

import (
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/json"
)

func streamGet(id string, v types.Value, w io.Writer) (bool, error) {
	err := json.ToJSON(v, w, json.ToOptions{
		Lists:  true,
		Maps:   true,
		Indent: "",
	})
	if err != nil {
		return false, fmt.Errorf("Key '%s' has non-Replicant data of type: %s", id, types.TypeOf(v).Describe())
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
