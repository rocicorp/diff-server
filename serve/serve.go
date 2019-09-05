package serve

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/spec"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/julienschmidt/httprouter"

	"github.com/aboodman/replicant/db"
	"github.com/aboodman/replicant/util/chk"
)

const (
	aws_access_key_id     = "REPLICANT_AWS_ACCESS_KEY_ID"
	aws_secret_access_key = "REPLICANT_AWS_SECRET_ACCESS_KEY"
	aws_region            = "us-west-2"
)

var (
	servers = map[string]*server{}
	sess    *session.Session
	mu      sync.Mutex
)

func Handler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(os.Stderr, "handler panicked: %+v\n", err)
			debug.PrintStack()
		}
	}()

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
	s.router.ServeHTTP(w, r)
}

func getServer(name string) (*server, error) {
	mu.Lock()
	defer mu.Unlock()

	s := servers[name]
	if s == nil {
		var cs chunks.ChunkStore
		if os.Getenv(aws_access_key_id) != "" {
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
		} else {
			td, err := ioutil.TempDir("", "")
			if err != nil {
				return nil, err
			}
			fmt.Println("No AWS credentials found in environment")
			fmt.Println("Running on local disk at: ", td)
			sp, err := spec.ForDatabase(td)
			if err != nil {
				return nil, err
			}
			cs = sp.NewChunkStore()
		}
		router := datas.Router(cs, "/serve/"+name)
		noms := datas.NewDatabase(cs)
		db, err := db.New(noms, "server")
		if err != nil {
			return nil, err
		}
		s = &server{router: router, db: db}
		servers[name] = s

		s.router.POST("/serve/"+name+"/sync", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			s.sync(w, req, ps)
		})
	}

	return s, nil
}

type server struct {
	router *httprouter.Router
	db     *db.DB
	mu     sync.Mutex
}

func (s *server) sync(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.db.Reload()
	if err != nil {
		serverError(w, err)
		return
	}
	params := req.URL.Query()
	clientHash, ok := hash.MaybeParse(params.Get("head"))
	if !ok {
		clientError(w, "invalid value for head param")
		return
	}
	var clientCommit db.Commit
	clientVal := s.db.Noms().ReadValue(clientHash)
	if clientVal == nil {
		clientError(w, "Specified hash not found")
		return
	}
	err = marshal.Unmarshal(clientVal, &clientCommit)
	if err != nil {
		clientError(w, "Invalid client commit")
		return
	}
	mergedCommit, err := db.HandleSync(s.db, clientCommit)
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, strings.NewReader(mergedCommit.TargetHash().String()))
}

func clientError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Println(http.StatusBadRequest, msg)
	io.Copy(w, strings.NewReader(msg))
}

func serverError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(os.Stderr, err.Error())
}
