package jsoms

import (
	"encoding/json"
	"errors"

	"github.com/attic-labs/noms/go/hash"
)

type Hash struct {
	hash.Hash
}

func (h Hash) MarshalJSON() ([]byte, error) {
	return []byte(h.Hash.String()), nil
}

func (h *Hash) UnmarshalJSON(data []byte) (err error) {
	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	hash, ok := hash.MaybeParse(str)
	if !ok {
		return errors.New("Invaild hash string")
	}
	h.Hash = hash
	return nil
}
