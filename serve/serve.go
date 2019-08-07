package serve

import (
	"net/http"
	"regexp"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/aboodman/replicant/util/chk"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	re, err := regexp.Compile("^/([^/]+)/(.*)")
	chk.NoError(err)
	parts := re.FindStringSubmatch(r.URL.Path)
	if parts == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid request"))
		return
	}
	dbName := parts[1]

	sess := session.Must(session.NewSession(aws.NewConfig().WithRegion("us-west-2")))
	cs := nbs.NewAWSStore("replicant", dbName, "aa-replicant", s3.New(sess), dynamodb.New(sess), 1<<28)
	router := datas.Router(cs, "/"+dbName)
	router.ServeHTTP(w, r)
}
