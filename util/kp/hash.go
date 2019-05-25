package kp

import (
	"errors"

	"github.com/attic-labs/noms/go/hash"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type HashValue hash.Hash

func (h *HashValue) Set(value string) error {
	v, ok := hash.MaybeParse(value)
	if !ok {
		return errors.New("Invalid hash string")
	}
	*h = (HashValue)(v)
	return nil
}

func (h *HashValue) String() string {
	return (*hash.Hash)(h).String()
}

func Hash(s kingpin.Settings, target *hash.Hash) {
	s.SetValue((*HashValue)(target))
	return
}
