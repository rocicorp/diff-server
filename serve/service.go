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

	rlog "github.com/aboodman/replicant/util/log"
	"github.com/attic-labs/noms/go/spec"
	"github.com/dgrijalva/jwt-go"
)

var (
	// /serve/<account>/<db>/<cmd>
	pathRegex = regexp.MustCompile(`^/serve/(\w+)/(\w+)/(\w+)/?$`)
)

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

	server, cErr, sErr := s.getServer(r)
	if cErr != "" {
		clientError(w, cErr)
		return
	}
	if sErr != nil {
		serverError(w, sErr)
		return
	}
	server.ServeHTTP(w, r)
}

func (s *Service) getServer(req *http.Request) (r *server, clientError string, serverError error) {
	match := pathRegex.FindStringSubmatch(req.URL.Path)
	if match == nil {
		return nil, fmt.Sprintf("Invalid request path"), nil
	}

	acc, db := match[1], match[2]
	if acc == "" {
		return nil, "account parameter is required", nil
	}
	if db == "" {
		return nil, "db parameter is required", nil
	}

	token := req.Header.Get("Authorization")
	err := checkAccess(acc, db, token, s.accounts)
	if err != nil {
		return nil, err.Error(), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", acc, db)
	r = s.servers[key]
	if r != nil {
		return r, "", nil
	}

	ss := s.storageRoot + "/" + key
	sp, err := spec.ForDatabase(ss)
	if err != nil {
		return nil, "", err
	}

	server, err := newServer(sp.NewChunkStore(), fmt.Sprintf("/serve/%s/%s", acc, db), "server")
	s.servers[key] = server
	if err != nil {
		return nil, "", err
	}
	return server, "", nil
}

// Claims are the JWT claims Replicant uses for authentication.
type Claims struct {
	jwt.StandardClaims
	DB string `json:"db,omitempty"`
}

func checkAccess(accountID, dbName, token string, accounts []Account) error {
	acc, err := lookupAccount(accountID, accounts)
	if err != nil {
		return err
	}

	// If account has no public key, it's publicly available.
	if acc.Pubkey == nil {
		return nil
	}

	if token == "" {
		return errors.New("authorization header is required")
	}

	var claims Claims
	_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		pk, err := jwt.ParseECPublicKeyFromPEM(acc.Pubkey)
		if err != nil {
			return nil, err
		}
		return pk, nil
	})
	if err != nil {
		return err
	}
	if claims.DB != dbName {
		return errors.New("token does not grant access to specified database")
	}
	return nil
}

func lookupAccount(accountID string, accounts []Account) (Account, error) {
	for _, a := range accounts {
		if a.ID == accountID {
			return a, nil
		}
	}
	return Account{}, fmt.Errorf("No such account: '%s'", accountID)
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
