package json

import (
	"encoding/json"

	"github.com/attic-labs/noms/go/spec"
)

type Spec struct {
	spec.Spec
}

func (s Spec) MarshalJSON() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *Spec) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	sp, err := spec.ForDatabase(str)
	if err != nil {
		return err
	}
	s.Spec = sp
	return nil
}
