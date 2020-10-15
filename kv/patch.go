package kv

// This file implements JSON Patch format for Noms-based Maps.
// See http://jsonpatch.com/
//
// Notes:
// - only currently supports the "add", "remove", and "replace" operations.
// - can only compute diffs on Noms values that are Boolean|Number|String, or Lists and Maps containing those types.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/types"
	"roci.dev/diff-server/util/chk"
	nomsjson "roci.dev/diff-server/util/noms/json"
)

const (
	// OpAdd is the JSONPatch "add" operation.
	OpAdd = "add"
	// OpRemove is the JSONPatch "add" operation.
	OpRemove = "remove"
	// OpReplace is the JSONPatch "replace" operation.
	OpReplace = "replace"
)

// Operation is a single JSONPatch change.
type Operation struct {
	Op          string          `json:"op"`
	Path        string          `json:"path"`
	Value       json.RawMessage `json:"value,omitempty"`
	ValueString string          `json:"valueString,omitempty"`
}

func jsonPointerEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

func jsonPointerUnescape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~1", "/"), "~0", "~")
}

// Diff calculates the difference between two maps as a JSON patch. Presently only
// creates ops at the top level, at the level of keys, so not super efficient.
func Diff(version uint32, from, to Map, r []Operation) ([]Operation, error) {
	dChan := make(chan types.ValueChanged)
	sChan := make(chan struct{})
	out := make(chan Operation)

	go func() {
		defer close(dChan)
		// Diffing is delegated to the underlying noms maps.
		to.NomsMap().Diff(from.NomsMap(), dChan, sChan)
	}()

	wg := &sync.WaitGroup{}
	var err error

	// We do this in parallel because ToJSON() below can end up requiring fetching more data, which we don't want
	// serialized.
	for i := 0; i < runtime.NumCPU()*2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range dChan {
				chk.Equal(types.StringKind, d.Key.Kind())

				op := Operation{
					Path: fmt.Sprintf("/%s", jsonPointerEscape(string(d.Key.(types.String)))),
				}
				switch d.ChangeType {
				case types.DiffChangeRemoved:
					op.Op = OpRemove
				case types.DiffChangeAdded, types.DiffChangeModified:
					b := &bytes.Buffer{}
					err = nomsjson.ToJSON(d.NewValue, b)
					if err != nil {
						// Would be nice to return an error out of here but there is no plumbing
						// for it. If you have time feel free.
						chk.Fail("Couldn't convert noms value to json: %#v", d)
					}
					if d.ChangeType == types.DiffChangeAdded {
						op.Op = OpAdd
					} else {
						op.Op = OpReplace
					}
					if version == 0 {
						op.Value = json.RawMessage(b.Bytes())
					} else {
						op.ValueString = string(b.Bytes())
					}
				default:
					chk.Fail("Unexpected ChangeType: %#v", d)
				}
				out <- op
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	for op := range out {
		r = append(r, op)
	}

	if err != nil {
		return nil, err
	}

	sort.Slice(r, func(i, j int) bool {
		return r[i].Path < r[j].Path
	})

	return r, nil
}

// ApplyPatch applies the given series of ops to the input Map.
func ApplyPatch(version uint32, vrw types.ValueReadWriter, to Map, patch []Operation) (Map, error) {
	if len(patch) == 0 {
		return to, nil
	}
	ed := to.Edit()
	for _, op := range patch {
		if version >= 2 &&
			op.Path == "" && op.Op == OpReplace && op.ValueString == "{}" {
			// Clear map
			emptyMap := NewMap(ed.noms)
			ed = emptyMap.Edit()
			continue
		}

		if !strings.HasPrefix(op.Path, "/") {
			return Map{}, fmt.Errorf("Invalid path %s - must start with /", op.Path)
		}
		p := types.String(jsonPointerUnescape(op.Path[1:]))
		switch op.Op {
		case OpAdd, OpReplace:
			var v types.Value
			var err error
			if version == 0 {
				v, err = nomsjson.FromJSON(op.Value, vrw)
			} else {
				v, err = nomsjson.FromJSON([]byte(op.ValueString), vrw)
			}
			if err != nil {
				return Map{}, fmt.Errorf("couldnt parse value from JSON '%s': %w", op.Value, err)
			}
			if err := ed.Set(p, v); err != nil {
				return Map{}, err
			}
		case OpRemove:
			if len(p) == 0 { // Remove("/")
				emptyMap := NewMap(ed.noms)
				ed = emptyMap.Edit()
			} else if err := ed.Remove(p); err != nil {
				return Map{}, err
			}
		default:
			return Map{}, fmt.Errorf("Unknown JSON Patch operation: %s", op.Op)
		}
	}
	return ed.Build(), nil
}
