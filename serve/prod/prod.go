// Package prod implements our top-level production server entrypoint for Zeit Now.
package prod

import (
	"net/http"
	"os"

	"github.com/aboodman/replicant/serve"
	"github.com/aboodman/replicant/serve/accounts"
	"github.com/attic-labs/noms/go/spec"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	aws_access_key_id     = "REPLICANT_AWS_ACCESS_KEY_ID"
	aws_secret_access_key = "REPLICANT_AWS_SECRET_ACCESS_KEY"
	aws_region            = "us-west-2"
)

var (
	svc = serve.NewService("aws:replicant/aa-replicant2", accounts.Accounts())
)

func init() {
	spec.GetAWSSession = func() *session.Session {
		return session.Must(session.NewSession(
			aws.NewConfig().WithRegion(aws_region).WithCredentials(
				// Have to do this wackiness because not allowed to set AWS env variables in Now for some reason.
				credentials.NewStaticCredentials(
					os.Getenv(aws_access_key_id),
					os.Getenv(aws_secret_access_key), ""))))
	}
}

// Handler implements the Zeit Now entrypoint for our server.
func Handler(w http.ResponseWriter, r *http.Request) {
	svc.ServeHTTP(w, r)
}
