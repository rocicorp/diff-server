package main

import (
	"log"
	"net/http"

	"github.com/aboodman/replicant/serve"
)

func main() {
	http.HandleFunc("/", serve.Handler)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
