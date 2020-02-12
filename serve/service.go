package serve

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/attic-labs/noms/go/spec"
	"github.com/dgrijalva/jwt-go"
	rlog "roci.dev/replicant/util/log"
)

var (
	// /<account>/<db>/<cmd>
	pathRegex = regexp.MustCompile(`^\/([\w-]+)\/([\w-]+)\/([\w-]+)\/?$`)
	origin    = "server"
)

func SetFakeOrigin(override string) func() {
	original := origin
	origin = override
	return func() {
		origin = original
	}
}

// Service is a running instance of the Replicant service. A service handles one or more servers.
type Service struct {
	storageRoot string
	urlPrefix   string
	accounts    []Account
	servers     map[string]*server
	mu          sync.Mutex
}

// Account is information about a customer of Replicant. This is a stand-in for what will eventually be
// an accounts database.
type Account struct {
	ID     string
	Name   string
	Pubkey []byte // PEM-formatted ECDSA public key
}

// NewService creates a new instances of the Replicant web service.
func NewService(storageRoot string, accounts []Account) *Service {
	return &Service{
		storageRoot: storageRoot,
		accounts:    accounts,
		servers:     map[string]*server{},
		mu:          sync.Mutex{},
	}
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rlog.Init(os.Stderr, rlog.Options{Prefix: true})

	server, cErr, authErr, sErr := s.getServer(r)
	if cErr != "" {
		clientError(w, http.StatusBadRequest, cErr)
		return
	}
	if authErr != nil {
		clientError(w, http.StatusForbidden, authErr.Error())
		return
	}
	if sErr != nil {
		serverError(w, sErr)
		return
	}
	server.ServeHTTP(w, r)
}

func (s *Service) getServer(req *http.Request) (r *server, clientError string, authError, serverError error) {
	match := pathRegex.FindStringSubmatch(req.URL.Path)
	if match == nil {
		return nil, fmt.Sprintf("Invalid request path"), nil, nil
	}

	acc, db := match[1], match[2]
	if acc == "" {
		return nil, "account parameter is required", nil, nil
	}
	if db == "" {
		return nil, "db parameter is required", nil, nil
	}

	token := req.Header.Get("Authorization")
	clientErr, authErr := checkAccess(acc, db, token, s.accounts)
	if clientErr != nil {
		return nil, clientErr.Error(), nil, nil
	}
	if authErr != nil {
		return nil, "", authErr, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", acc, db)
	r = s.servers[key]
	if r != nil {
		err := r.db.Reload()
		if err != nil {
			return nil, "", nil, err
		}
		return r, "", nil, nil
	}

	ss := s.storageRoot + "/" + key
	sp, err := spec.ForDatabase(ss)
	if err != nil {
		return nil, "", nil, err
	}

	server, err := newServer(sp.NewChunkStore(), fmt.Sprintf("/%s/%s", acc, db), origin)
	s.servers[key] = server
	if err != nil {
		return nil, "", nil, err
	}
	return server, "", nil, nil
}

// Claims are the JWT claims Replicant uses for authentication.
type Claims struct {
	jwt.StandardClaims
	DB string `json:"db,omitempty"`
}

func checkAccess(accountID, dbName, token string, accounts []Account) (clientError, authError error) {
	acc, ok := lookupAccount(accountID, accounts)
	if !ok {
		return fmt.Errorf("No such account: '%s'", accountID), nil
	}

	// If account has no public key, it's publicly available.
	if acc.Pubkey == nil {
		return nil, nil
	}

	if token == "" {
		return nil, errors.New("Authorization header is required")
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
		return nil, fmt.Errorf("Invalid JWT: %s", err.Error())
	}
	if claims.DB != dbName {
		return nil, errors.New("Token does not grant access to specified database")
	}
	return nil, nil
}

func lookupAccount(accountID string, accounts []Account) (acc Account, ok bool) {
	for _, a := range accounts {
		if a.ID == accountID {
			return a, true
		}
	}
	return Account{}, false
}

func clientError(w http.ResponseWriter, code int, body string) {
	w.WriteHeader(code)
	log.Printf("Client error: HTTP %d: %s", code, body)
	io.Copy(w, strings.NewReader(body))
}

func serverError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	log.Println(err.Error())
}
