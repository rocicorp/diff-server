// Package prod implements our top-level production server entrypoint for Zeit Now.
package prod

import (
	"net/http"
	"os"

	"github.com/attic-labs/noms/go/spec"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/mux"
	zl "github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"roci.dev/diff-server/serve"
	"roci.dev/diff-server/serve/accounts"
	"roci.dev/diff-server/util/loghttp"
)

const (
	aws_access_key_id     = "REPLICANT_AWS_ACCESS_KEY_ID"
	aws_secret_access_key = "REPLICANT_AWS_SECRET_ACCESS_KEY"
	aws_region            = "us-west-2"
)

var (
	handler            http.Handler
	headerLogWhitelist = []string{"Authorization", "Content-Type", "Host", "X-Replicache-SyncID"}
)

func init() {
	// Zeit now has a 4kb log limit per request, so set up some aggressive HTTP log filters.
	loghttp.Filters = append(loghttp.Filters, loghttp.NewBodyElider(500).Filter)
	loghttp.Filters = append(loghttp.Filters, loghttp.NewHeaderWhitelist(headerLogWhitelist).Filter)

	zl.SetGlobalLevel(zl.DebugLevel)
	zlog.Logger = zlog.Output(zl.ConsoleWriter{Out: os.Stderr, TimeFormat: "02 Jan 06 15:04:05.000 -0700", NoColor: true})
	spec.GetAWSSession = func() *session.Session {
		return session.Must(session.NewSession(
			aws.NewConfig().WithRegion(aws_region).WithCredentials(
				// Have to do this wackiness because not allowed to set AWS env variables in Now for some reason.
				credentials.NewStaticCredentials(
					os.Getenv(aws_access_key_id),
					os.Getenv(aws_secret_access_key), ""))))
	}

	svc := serve.NewService("aws:replicant/aa-replicant2", accounts.Accounts(), "", serve.ClientViewGetter{}, false)
	mux := mux.NewRouter()
	serve.RegisterHandlers(svc, mux)
	handler = mux
}

// Handler implements the Zeit Now entrypoint for our server.
func Handler(w http.ResponseWriter, r *http.Request) {
	handler.ServeHTTP(w, r)
}
