package serve

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/aboodman/replicant/util/chk"
)

const (
	aws_access_key_id     = "REPLICANT_AWS_ACCESS_KEY_ID"
	aws_secret_access_key = "REPLICANT_AWS_SECRET_ACCESS_KEY"
	aws_region            = "us-west-2"
)

var (
	servers = map[string]*Server{}
	sess    *session.Session
	mu      sync.Mutex
)

// Handler implements the Zeit Now entrypoint for our server.
func Handler(w http.ResponseWriter, r *http.Request) {
	re, err := regexp.Compile("^/serve/([^/]+)/(.*)")
	chk.NoError(err)

	parts := re.FindStringSubmatch(r.URL.Path)
	if parts == nil {
		clientError(w, "invalid database name")
		return
	}
	dbName := parts[1]
	s, err := getServer(dbName)
	if err != nil {
		serverError(w, err)
	}
	s.ServeHTTP(w, r)
}

func getServer(name string) (*Server, error) {
	mu.Lock()
	defer mu.Unlock()

	s := servers[name]
	if s != nil {
		return s, nil
	}

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
	fmt.Printf("Found AWS credentials in environment. Running against DynamoDB table: %s, bucket: %s, namespace: %s\n", table, bucket, name)
	var err error
	s, err = NewServer(cs, "/serve/"+name, "server")
	servers[name] = s
	if err != nil {
		return nil, err
	}
	return s, nil
}
