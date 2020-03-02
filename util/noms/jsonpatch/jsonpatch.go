// Package jsonpatch implements the JSON Patch format for Noms.
// See http://jsonpatch.com/
//
// Notes:
// - jsonpatch only currently supports the "add", "remove", and "replace" operations.
// - jsonpatch can only compute diffs on Noms values that are Boolean|Number|String, or Lists and Maps containing those types.
package jsonpatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/types"
	nomsjson "github.com/attic-labs/noms/go/util/json"
	"roci.dev/diff-server/util/chk"
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
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

func jsonPointerEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

func jsonPointerUnescape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~1", "/"), "~0", "~")
}

// Diff calculates the difference between two Noms maps as a JSON patch and streams the result to the provided writer.
func Diff(from, to types.Map, r []Operation) ([]Operation, error) {
	dChan := make(chan types.ValueChanged)
	sChan := make(chan struct{})
	out := make(chan Operation)

	go func() {
		defer close(dChan)
		to.Diff(from, dChan, sChan)
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
					err = nomsjson.ToJSON(d.NewValue, b, nomsjson.ToOptions{
						Lists: true,
						Maps:  true,
					})
					// Danger: swallowed error.
					if err != nil {
						return
					}
					if d.ChangeType == types.DiffChangeAdded {
						op.Op = OpAdd
					} else {
						op.Op = OpReplace
					}
					op.Value = json.RawMessage(b.Bytes())
				default:
					chk.Fail("Unexpected Noms ChangeType: %#v", d)
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

func ApplyOne(noms types.ValueReadWriter, onto *types.MapEditor, op Operation) error {
	if !strings.HasPrefix(op.Path, "/") {
		return fmt.Errorf("Invalid path %s - must start with /", op.Path)
	}
	p := types.String(jsonPointerUnescape(op.Path[1:]))
	switch op.Op {
	case OpAdd, OpReplace:
		v, err := nomsjson.FromJSON(bytes.NewReader([]byte(op.Value)), noms, nomsjson.FromOptions{})
		if err != nil {
			return err
		}
		onto.Set(p, v)
	case OpRemove:
		onto.Remove(p)
	default:
		return fmt.Errorf("Unknown JSON Patch operation: %s", op.Op)
	}
	return nil
}

func Apply(noms types.ValueReadWriter, onto types.Map, patch []Operation) (types.Map, error) {
	if len(patch) == 0 {
		return onto, nil
	}
	ed := onto.Edit()
	for _, op := range patch {
		err := ApplyOne(noms, ed, op)
		if err != nil {
			return types.Map{}, err
		}
	}
	return ed.Map(), nil
}
