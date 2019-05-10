package main

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
