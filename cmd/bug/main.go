package main

import (
	"fmt"

	"github.com/attic-labs/noms/go/chunks"
	//"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
)

type Foo struct {
	Bar types.Ref
	Baz types.String
}

func main() {
	ts := &chunks.TestStorage{}
	vs := types.NewValueStore(ts.NewView())

	// This is how I discovered this bug - very easy mistake to make
	// note missing `noms:",omitempty"` on `Bar` field.
	//
	// nf := marshal.MustMarshal(vs, Foo{Baz:"monkey"}).(types.Struct)
	// fmt.Println(nf.Get("bar"))
	//
	// ... but you can also get the bug directly via the types API:

	foo := types.NewStruct("", types.StructData{
		"bar": types.Ref{},
	})
	c := types.EncodeValue(foo)
	v := types.DecodeValue(c, vs).(types.Struct)
	fmt.Println(v.Get("bar"))
}

