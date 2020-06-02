package serve

import (
	"fmt"
	"net/http"

	"roci.dev/diff-server/util/version"
)

// hello prints a hello message to let users know the server is running.
func (s *Service) hello(w http.ResponseWriter, r *http.Request) {
	l := logger(r)
	if r.Method != "GET" {
		unsupportedMethodError(w, r.Method, l)
		return
	}
	w.Header().Add("Content-type", "text/plain")
	w.Write([]byte("Hello from Replicache\n"))
	w.Write([]byte(fmt.Sprintf("Version: %s\n", version.Version())))
}
