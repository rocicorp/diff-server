package api

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

	"roci.dev/diff-server/account"
	"roci.dev/diff-server/serve"
	"roci.dev/diff-server/util/loghttp"
)

const (
	aws_access_key_id     = "REPLICANT_AWS_ACCESS_KEY_ID"
	aws_secret_access_key = "REPLICANT_AWS_SECRET_ACCESS_KEY"
	aws_region            = "us-west-2"

	storageRoot = "aws:replicant/aa-replicant2"
)

var (
	diffServiceHandler http.Handler
	headerLogAllowlist = []string{"Authorization", "Content-Type", "Host", "X-Replicache-SyncID"}
)

func init() {
	// Zeit now has a 4kb log limit per request, so set up some aggressive HTTP log filters.
	loghttp.Filters = append(loghttp.Filters, loghttp.NewBodyElider(500).Filter)
	loghttp.Filters = append(loghttp.Filters, loghttp.NewHeaderAllowlist(headerLogAllowlist).Filter)

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

	accountDB, err := account.NewDB(storageRoot)
	if err != nil {
		panic(err)
	}

	svc := serve.NewService(storageRoot, account.MaxASClientViewHosts, accountDB, false, serve.ClientViewGetter{}, false)
	mux := mux.NewRouter()
	serve.RegisterHandlers(svc, mux)
	diffServiceHandler = mux
}

// DiffServiceHandler implements the Vercel entrypoint for the DiffService.
func DiffServiceHandler(w http.ResponseWriter, r *http.Request) {
	diffServiceHandler.ServeHTTP(w, r)
}
