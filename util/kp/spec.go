package kp

import (
	"github.com/attic-labs/noms/go/spec"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type SpecValue spec.Spec

func (s *SpecValue) Set(value string) error {
	sp, err := spec.ForDatabase(value)
	if err != nil {
		return err
	}
	*s = (SpecValue)(sp)
	return nil
}

func (s *SpecValue) String() string {
	return (*spec.Spec)(s).String()
}

func DatabaseSpec(s kingpin.Settings) (target *spec.Spec) {
	target = &spec.Spec{}
	s.SetValue((*SpecValue)(target))
	return
}
