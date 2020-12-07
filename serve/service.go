package serve

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"

	"roci.dev/diff-server/db"
	servetypes "roci.dev/diff-server/serve/types"

	zl "github.com/rs/zerolog"

	// Log all HTTP requests

	_ "roci.dev/diff-server/util/loghttp"
)

var (
	// /<account>/<db>/<cmd>
	pathRegex = regexp.MustCompile(`^\/([\w-]+)\/([\w-]+)\/([\w-]+)\/?$`)
)

// Service is an instance of the Replicache Diffserver services.
type Service struct {
	storageRoot          string
	urlPrefix            string
	accounts             []Account
	nomsen               map[string]datas.Database
	overridClientViewURL string // Overrides account client view URL, eg for testing.
	enableInject         bool
	mu                   sync.Mutex

	// cvg may be nil, in which case the server skips the client view request in pull, which is
	// useful if you are populating the db directly or in tests.
	clientViewGetter clientViewGetter
}

type clientViewGetter interface {
	Get(url string, req servetypes.ClientViewRequest, authToken string, syncID string) (servetypes.ClientViewResponse, int, error)
}

// Account represents a regular customer account (as opposed to an auto-signup account). 
// This is a temporary stand-in that will be replaced by the account service.
type Account struct {
	ID            string
	Name          string
	Pubkey        []byte // PEM-formatted ECDSA public key
	ClientViewURL string
}

// NewService creates a new instances of the Replicant web service.
func NewService(storageRoot string, accounts []Account, overrideClientViewURL string, cvg clientViewGetter, enableInject bool) *Service {
	return &Service{
		storageRoot:          storageRoot,
		accounts:             accounts,
		nomsen:               map[string]datas.Database{},
		overridClientViewURL: overrideClientViewURL,
		enableInject:         enableInject,
		mu:                   sync.Mutex{},
		clientViewGetter:     cvg,
	}
}

// RegisterHandlers register's Service's handlers on the given router.
func RegisterHandlers(s *Service, router *mux.Router) {
	router.SkipClean(true)
	router.HandleFunc("/", s.hello)
	inject := alice.New(panicCatcher).ThenFunc(s.inject)
	router.Handle("/inject", inject)
	pull := alice.New(contextLogger, panicCatcher, logHTTP).ThenFunc(s.pull)
	router.Handle("/pull", pull)
}

func (s *Service) GetDB(accountID, clientID string) (*db.DB, error) {
	noms, err := s.getNoms(accountID)
	if err != nil {
		return nil, err
	}
	dsName := fmt.Sprintf("client/%s", clientID)
	db, err := db.New(noms.GetDataset(dsName))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (s *Service) getNoms(accountID string) (datas.Database, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := s.nomsen[accountID]
	if n == nil {
		sp, err := spec.ForDatabase(fmt.Sprintf("%s/%s", s.storageRoot, accountID))
		if err != nil {
			return nil, err
		}
		n = sp.GetDatabase()
		s.nomsen[accountID] = n
	} else {
		n.Rebase()
	}
	return n, nil
}

func lookupAccount(accountID string, accounts []Account) (acc Account, ok bool) {
	for _, a := range accounts {
		if a.ID == accountID {
			return a, true
		}
	}
	return Account{}, false
}

func unsupportedMethodError(w http.ResponseWriter, m string, l zl.Logger) {
	clientError(w, http.StatusMethodNotAllowed, fmt.Sprintf("Unsupported method: %s", m), l)
}

func clientError(w http.ResponseWriter, code int, body string, l zl.Logger) {
	w.WriteHeader(code)
	l.Info().Int("status", code).Msg(body)
	io.Copy(w, strings.NewReader(body))
}

func serverError(w http.ResponseWriter, err error, l zl.Logger) {
	w.WriteHeader(http.StatusInternalServerError)
	l.Info().Int("status", http.StatusInternalServerError).Err(err).Send()
}
