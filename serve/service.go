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
	"roci.dev/diff-server/util/chk"
	rlog "roci.dev/diff-server/util/log"
	"roci.dev/diff-server/util/version"
)

var (
	// /<account>/<db>/<cmd>
	pathRegex = regexp.MustCompile(`^\/([\w-]+)\/([\w-]+)\/([\w-]+)\/?$`)
)

// Service is a running instance of the Replicant service. A service handles one or more servers.
type Service struct {
	storageRoot          string
	urlPrefix            string
	accounts             []Account
	servers              map[string]*server
	overridClientViewURL string // Overrides account client view URL, eg for testing.
	mu                   sync.Mutex
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
func NewService(storageRoot string, accounts []Account, clientViewURL string) *Service {
	return &Service{
		storageRoot:          storageRoot,
		accounts:             accounts,
		servers:              map[string]*server{},
		overridClientViewURL: clientViewURL,
		mu:                   sync.Mutex{},
	}
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rlog.Init(os.Stderr, rlog.Options{Prefix: true})

	if r.URL.Path == "/" {
		w.Header().Add("Content-type", "text/plain")
		w.Write([]byte("Hello from Replicache\n"))
		w.Write([]byte(fmt.Sprintf("Version: %s\n\n", version.Version())))
		w.Write([]byte("This is the root of the service.\n"))
		w.Write([]byte("To access an individual DB, try: /<accountid>/<dbname>/<cmd>\n"))
		return
	}

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
	account, clientErr, authErr := checkAccess(acc, db, token, s.accounts)
	if clientErr != nil {
		return nil, clientErr.Error(), nil, nil
	}
	if authErr != nil {
		return nil, "", authErr, nil
	}
	chk.NotNil(account)

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

	clientViewURL := account.ClientViewURL
	if s.overridClientViewURL != "" {
		log.Printf("WARNING: overriding all client view URLs with %s", s.overridClientViewURL)
		clientViewURL = s.overridClientViewURL
	}
	var cvg clientViewGetter
	if clientViewURL != "" {
		cvg = ClientViewGetter{url: clientViewURL}
	}
	server, err := newServer(sp.NewChunkStore(), fmt.Sprintf("/%s/%s", acc, db), cvg)
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

func clientError(w http.ResponseWriter, code int, body string) {
	w.WriteHeader(code)
	log.Printf("Client error: HTTP %d: %s", code, body)
	io.Copy(w, strings.NewReader(body))
}

func serverError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	log.Println(err.Error())
}
