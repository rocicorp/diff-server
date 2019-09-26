// Package prod implements our top-level production server entrypoint for Zeit Now.
package prod

import (
	"net/http"

	"github.com/aboodman/replicant/serve"
)

var (
	svc = serve.NewService("aws:replicant/aa-replicant2", "/serve")
)

// Handler implements the Zeit Now entrypoint for our server.
func Handler(w http.ResponseWriter, r *http.Request) {
	svc.ServeHTTP(w, r)
}
