// Package prod implements our top-level production server entrypoint for Zeit Now.
package prod

import (
	"log"
	"net/http"
	"os"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/aboodman/replicant/serve"
	"github.com/aboodman/replicant/util/chk"
)

const (
	aws_access_key_id     = "REPLICANT_AWS_ACCESS_KEY_ID"
	aws_secret_access_key = "REPLICANT_AWS_SECRET_ACCESS_KEY"
	aws_region            = "us-west-2"
)

var (
	sess       *session.Session
	awsService = serve.NewServiceWithFactory("/serve/", awsChunkStore)
)

// Handler implements the Zeit Now entrypoint for our server.
func Handler(w http.ResponseWriter, r *http.Request) {
	awsService.ServeHTTP(w, r)
}

func awsChunkStore(name string) (chunks.ChunkStore, error) {
	var cs chunks.ChunkStore
	if os.Getenv(aws_access_key_id) == "" {
		chk.Fail("Cannot create server - no aws credentials in environment")
	}
	if sess == nil {
		sess = session.Must(session.NewSession(
			aws.NewConfig().WithRegion(aws_region).WithCredentials(
				credentials.NewStaticCredentials(
					os.Getenv(aws_access_key_id),
					os.Getenv(aws_secret_access_key), ""))))
	}
	const table = "replicant"
	const bucket = "aa-replicant2"
	cs = nbs.NewAWSStore(table, name, bucket, s3.New(sess), dynamodb.New(sess), 1<<28)
	log.Printf("Found AWS credentials in environment. Running against DynamoDB table: %s, bucket: %s, namespace: %s", table, bucket, name)
	return cs, nil
}
