package serve

import (
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/julienschmidt/httprouter"

	"github.com/aboodman/replicant/util/chk"
)

type server struct {
	router *httprouter.Router
}

var (
	servers = map[string]*server{}
	sess    *session.Session
	mu      sync.Mutex
)

func Handler(w http.ResponseWriter, r *http.Request) {
	re, err := regexp.Compile("^/serve/([^/]+)/(.*)")
	chk.NoError(err)

	parts := re.FindStringSubmatch(r.URL.Path)
	if parts == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid request"))
		return
	}
	dbName := parts[1]
	getServer(dbName).router.ServeHTTP(w, r)
}

func getServer(name string) *server {
	mu.Lock()
	defer mu.Unlock()

	if sess == nil {
		sess = session.Must(session.NewSession(
			aws.NewConfig().WithRegion("us-west-2").WithCredentials(
				credentials.NewStaticCredentials(
					os.Getenv("REPLICANT_AWS_ACCESS_KEY_ID"),
					os.Getenv("REPLICANT_AWS_SECRET_ACCESS_KEY"), ""))))
	}

	s := servers[name]
	if s == nil {
		cs := nbs.NewAWSStore("replicant", name, "aa-replicant2", s3.New(sess), dynamodb.New(sess), 1<<28)
		router := datas.Router(cs, "/serve/"+name)
		s = &server{router: router}
		servers[name] = s
	}
	return s
}
