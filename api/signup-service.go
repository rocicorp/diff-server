package api

import (
	"html/template"
	"net/http"
	"os"

	"github.com/attic-labs/noms/go/spec"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/mux"
	zl "github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"roci.dev/diff-server/serve/signup"
	"roci.dev/diff-server/util/log"
)

var (
	signupHandler http.Handler
)

func init() {
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

	// TODO should probably be sharing a mux with DiffService.
	mux := mux.NewRouter()

	// Set up signup service.
	tmpl := template.Must(signup.ParseTemplates(signup.Templates()))
	service := signup.NewService(log.Default(), tmpl, storageRoot)
	signup.RegisterHandlers(service, mux)

	signupHandler = mux
}

// SignupHandler implements the Vercel entrypoint for the signup service.
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	signupHandler.ServeHTTP(w, r)
}
