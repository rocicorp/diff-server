// Copyright 2019 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package json

import (
	"fmt"
	"io"

	"github.com/attic-labs/noms/go/types"
	cjson "github.com/gibson042/canonicaljson-go"
)

// noNewlineWriter writes to an underlying io.Writer, omitting any trailing newline.
type noNewlineWriter struct {
	w io.Writer
}

// Helper broken out for testing.
func hasNewline(s string) bool {
	for _, runeValue := range s {
		if string(runeValue) == "\n" {
			return true
		}
	}
	return false
}

// Write implements the io.Writer interface.
func (w *noNewlineWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return w.w.Write(p)
	}

	var trailingNewline bool
	// Note: canonical json never has internal newlines.
	if hasNewline(string(p[len(p)-1])) {
		trailingNewline = true
		p = p[:len(p)-1]
	}
	n, err := w.w.Write(p)
	if trailingNewline && n == len(p) {
		n = n + 1
	}
	return n, err
}

// Canonicalize round-trips the json to canonicalize it.
func Canonicalize(JSON []byte) ([]byte, error) {
	var v interface{}
	if err := cjson.Unmarshal(JSON, &v); err != nil {
		return nil, fmt.Errorf("couldnt parse value '%s' as json: %w", string(JSON), err)
	}
	return cjson.Marshal(v)
}

// ToJSON encodes a Noms value as canonical JSON.
// It would be nice to have an option like the original noms
// ops.Indent which would enable pretty printing via the default json library.
func ToJSON(v types.Value, w io.Writer) error {
	// TODO: This is a quick hack that is expedient. We should marshal directly to the writer without
	// allocating a bunch of Go values.
	p, err := toPile(v)
	if err != nil {
		return err
	}

	enc := cjson.NewEncoder(&noNewlineWriter{w})
	return enc.Encode(p)
}

func toPile(v types.Value) (ret interface{}, err error) {
	switch v := v.(type) {
	case types.Bool:
		return bool(v), nil
	case types.Number:
		return float64(v), nil
	case types.String:
		return string(v), nil
	case types.Struct:
		if !Null().Equals(v) {
			return nil, fmt.Errorf("Unsupported struct type: %s", types.TypeOf(v).Describe())
		}
		return nil, nil
	case types.Map:
		r := make(map[string]interface{}, v.Len())
		v.Iter(func(k, cv types.Value) (stop bool) {
			sk, ok := k.(types.String)
			if !ok {
				err = fmt.Errorf("Map key kind %s not supported", types.KindToString[k.Kind()])
				return true
			}
			var cp interface{}
			cp, err = toPile(cv)
			if err != nil {
				return true
			}
			r[string(sk)] = cp
			return false
		})
		return r, err
	case types.List:
		r := make([]interface{}, v.Len())
		v.Iter(func(cv types.Value, i uint64) (stop bool) {
			var cp interface{}
			cp, err = toPile(cv)
			if err != nil {
				return true
			}
			r[i] = cp
			return false
		})
		return r, err
	}
	return nil, fmt.Errorf("Unsupported kind: %s", types.KindToString[v.Kind()])
}
