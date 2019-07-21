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
	// TODO: log this bug at noms
	// basically: an empty/zero ref gets encoded to an empty buffer
	// typically it's hard to cause this to happen but easy with marshaling, and the API does allow it  (see below)
	// we should do ... something when encoding an empty noms value.
	// either reject it and fail the encode, or encode a meaningful zero value, or something.
	ts := &chunks.TestStorage{}
	vs := types.NewValueStore(ts.NewView())

	//nf := marshal.MustMarshal(vs, Foo{Baz:"monkey"}).(types.Struct)
	//fmt.Println(nf.Get("bar"))

	foo := types.NewStruct("", types.StructData{
		"bar": types.Ref{},
	})
	c := types.EncodeValue(foo)
	v := types.DecodeValue(c, vs).(types.Struct)
	fmt.Println(v.Get("bar"))
}

