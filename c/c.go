package main

// compile like:
// go build -buildmode=c-archive -o rep.a c.go
//
// then see instructions in rep.c

import (
	"C"
	"fmt"

	"github.com/attic-labs/noms/go/types"
)

//export SayHello
func SayHello() {
	s := types.String("Hello, from Noms!")
	fmt.Println(s, s.Hash().String())
}

func main() {

}
