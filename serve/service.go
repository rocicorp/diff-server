package serve

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/aboodman/replicant/util/chk"
	rlog "github.com/aboodman/replicant/util/log"
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/spec"
)

type ChunkStoreFactory func(name string) (chunks.ChunkStore, error)

// Service is a running instance of the Replicant service. A service handles one or more servers.
type Service struct {
	prefix  string
	servers map[string]*server
	mu      sync.Mutex
	factory ChunkStoreFactory
}

func NewService(prefix, devPath string) *Service {
	chk.False(devPath == "", "devPath must be non-empty")
	return NewServiceWithFactory(prefix, localFactory(devPath))
}

func NewServiceWithFactory(prefix string, factory ChunkStoreFactory) *Service {
	return &Service{
		prefix:  prefix,
		servers: map[string]*server{},
		mu:      sync.Mutex{},
		factory: factory,
	}
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rlog.Init(os.Stderr, rlog.Options{Prefix: true})

	re, err := regexp.Compile("^" + regexp.QuoteMeta(s.prefix) + "([^/]+)/(.*)")
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

	cs, err := s.factory(name)
	if err != nil {
		return nil, err
	}

	server, err := newServer(cs, s.prefix+name, "server")
	s.servers[name] = server
	if err != nil {
		return nil, err
	}
	return server, nil
}

func localFactory(p string) ChunkStoreFactory {
	return func(name string) (chunks.ChunkStore, error) {
		fp := path.Join(p, name)
		sp, err := spec.ForDatabase(fp)
		if err != nil {
			return nil, err
		}
		err = os.MkdirAll(fp, 0755)
		if err != nil {
			return nil, err
		}
		return sp.NewChunkStore(), nil
	}
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
