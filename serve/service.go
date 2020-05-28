package serve

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/dgrijalva/jwt-go"

	"roci.dev/diff-server/db"
	servetypes "roci.dev/diff-server/serve/types"

	zl "github.com/rs/zerolog"

	// Log all HTTP requests
	"roci.dev/diff-server/util/log"
	_ "roci.dev/diff-server/util/loghttp"
)

var (
	// /<account>/<db>/<cmd>
	pathRegex = regexp.MustCompile(`^\/([\w-]+)\/([\w-]+)\/([\w-]+)\/?$`)
)

// Service is a running instance of the Replicant service.
type Service struct {
	storageRoot          string
	urlPrefix            string
	accounts             []Account
	nomsen               map[string]datas.Database
	overridClientViewURL string // Overrides account client view URL, eg for testing.
	enableInject         bool
	mu                   sync.Mutex

	reqID uint64

	// cvg may be nil, in which case the server skips the client view request in pull, which is
	// useful if you are populating the db directly or in tests.
	clientViewGetter clientViewGetter
}

type clientViewGetter interface {
	Get(url string, req servetypes.ClientViewRequest, authToken string, syncID string) (servetypes.ClientViewResponse, int, error)
}

// Account is information about a customer of Replicant. This is a stand-in for what will eventually be
// an accounts database.
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

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := log.Default().With().Str("req", r.URL.String()).Uint64("rid", atomic.AddUint64(&s.reqID, 1))
	syncID := r.Header.Get("X-Replicache-SyncID")
	if syncID != "" {
		c = c.Str("syncID", syncID)
	}
	l := c.Logger()
	l.Info().Msg("received request")

	defer func() {
		err := recover()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			l.Error().Msgf("Handler panicked: %#v", err)
		}
	}()

	switch r.URL.Path {
	case "/":
		s.hello(w, r, l)
	case "/pull":
		s.pull(w, r, l)
	case "/inject":
		s.inject(w, r, l)
	default:
		clientError(w, http.StatusNotFound, "not found", l)
	}
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

// Claims are the JWT claims Replicant uses for authentication.
type Claims struct {
	jwt.StandardClaims
	DB string `json:"db,omitempty"`
}

func checkAccess(accountID, dbName, token string, accounts []Account) (*Account, error, error) {
	acc, ok := lookupAccount(accountID, accounts)
	if !ok {
		return nil, fmt.Errorf("No such account: '%s'", accountID), nil
	}

	// If account has no public key, it's publicly available.
	if acc.Pubkey == nil {
		return &acc, nil, nil
	}

	if token == "" {
		return nil, nil, errors.New("Authorization header is required")
	}

	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		pk, err := jwt.ParseECPublicKeyFromPEM(acc.Pubkey)
		if err != nil {
			return nil, fmt.Errorf("Invalid JWT: %s", err.Error())
		}
		return pk, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("Invalid JWT: %s", err.Error())
	}
	if claims.DB != dbName {
		return nil, nil, errors.New("Token does not grant access to specified database")
	}
	return &acc, nil, nil
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
