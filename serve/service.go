package serve

import (
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/aboodman/replicant/util/chk"
	rlog "github.com/aboodman/replicant/util/log"
	"github.com/attic-labs/noms/go/spec"
)

// Service is a running instance of the Replicant service. A service handles one or more servers.
type Service struct {
	storageRoot string
	urlPrefix   string
	servers     map[string]*server
	mu          sync.Mutex
}

func NewService(storageRoot, urlPrefix string) *Service {
	return &Service{
		storageRoot: storageRoot,
		urlPrefix:   urlPrefix,
		servers:     map[string]*server{},
		mu:          sync.Mutex{},
	}
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rlog.Init(os.Stderr, rlog.Options{Prefix: true})

	re, err := regexp.Compile("^" + regexp.QuoteMeta(s.urlPrefix) + "/([^/]+)/(.*)")
	chk.NoError(err)

	parts := re.FindStringSubmatch(r.URL.Path)
	if parts == nil {
		clientError(w, "invalid database name")
		return
	}
	dbName := parts[1]
	server, err := s.getServer(dbName)
	if err != nil {
		serverError(w, err)
	}
	server.ServeHTTP(w, r)
}

func (s *Service) getServer(name string) (*server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := s.servers[name]
	if r != nil {
		return r, nil
	}

	sp, err := spec.ForDatabase(s.storageRoot + "/" + name)
	if err != nil {
		return nil, err
	}

	server, err := newServer(sp.NewChunkStore(), s.urlPrefix+"/"+name, "server")
	s.servers[name] = server
	if err != nil {
		return nil, err
	}
	return server, nil
}

func clientError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	log.Println(http.StatusBadRequest, msg)
	io.Copy(w, strings.NewReader(msg))
}

func serverError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	log.Println(err.Error())
}
