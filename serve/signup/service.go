package signup

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	zl "github.com/rs/zerolog"
	"roci.dev/diff-server/account"
)

// Templates returns the list of Templates the signup service needs.
// Add new templates to this list!
func Templates() []Template {
	return []Template{
		{Name: GetTemplateName, Content: GetTemplate},
		{Name: PostTemplateName, Content: PostTemplate},
	}
}

// Template contains a string with the template content. Normally we'd
// have the content in a file but I can't figure out how to access files
// at runtime with Vercel.
type Template struct {
	Name    string
	Content string
}

func ParseTemplates(templates []Template) (t *template.Template, err error) {
	t = template.New("")

	for _, tmpl := range templates {
		if _, err = t.New(tmpl.Name).Parse(tmpl.Content); err != nil {
			return
		}
	}

	return
}

// Service is an instance of the signup service. It returns a little form
// to fill out with account information, accepts a POST from the form, and creates
// the account in an account.DB.
type Service struct {
	logger      zl.Logger
	tmpl        *template.Template
	storageRoot string
}

// NewService instantiates the signup service. Handlers need to be registered with
// RegisterHandlers.
// TODO NewService should probably take an account.DB instead of its storageroot
func NewService(logger zl.Logger, tmpl *template.Template, storageRoot string) *Service {
	return &Service{logger, tmpl, storageRoot}
}

// Path is the URL path at which to serve. It is used when running locally.
// When running on Vercel we serve from under /api/, but there is a rewrite
// rule in now.json that maps /signup to the service api path.
const Path = "/signup"

// RegisterHandlers registers Service's handlers on the given router.
func RegisterHandlers(s *Service, router *mux.Router) {
	router.HandleFunc(Path, s.handle)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	// TODO account authorized clientview URLs should match on domain
	// TODO better error messages for errors in POST
	// TODO light form validation eg missing email
	// TODO retry if concurrent POSTs step on each other + test
	// TODO retry if saving new clientviewurl fails
	// TODO logging
	// TODO cache account.Records, eg only re-read every N second
	// TODO rate limiting
	// TODO add more text/explanation to POST template (currently just has the ID)
	// TODO figure out how to recommend they contact us and include in templates
	// TODO purty
	// TODO update setup instructions and docs to point to this service
	// TODO update diffs instructions to include account-db and template-path
	// TODO wipe prod account db to clean slate when we launch this feature
	//      (ie, remove the noise from us tire-kicking)

	if r.Method == "GET" {
		if err := s.tmpl.ExecuteTemplate(w, GetTemplateName, getTemplateArgs{GetTemplateNameField, GetTemplateEmailField}); err != nil {
			serverError(w, err, s.logger)
		}
		return

	} else if r.Method == "POST" {
		name := r.FormValue(GetTemplateNameField)
		email := r.FormValue(GetTemplateEmailField)
		// See TODOs above: we need light validation and better error messages
		db, err := account.NewDB(s.storageRoot)
		if err != nil {
			serverError(w, err, s.logger)
			return
		}
		accounts, err := account.ReadAllRecords(db)
		if err != nil {
			serverError(w, err, s.logger)
			return
		}
		id := accounts.NextASID
		accounts.Record[id] = account.Record{
			ID:          id,
			Name:        name,
			Email:       email,
			DateCreated: time.Now().String(),
		}
		accounts.NextASID++
		if err := account.WriteRecords(db, accounts); err != nil {
			// See TODOs above: retry if head was changed from under us.
			serverError(w, err, s.logger)
			return
		}
		templateArgs := postTemplateArgs{ID: fmt.Sprintf("%d", id)}
		if err := s.tmpl.ExecuteTemplate(w, PostTemplateName, templateArgs); err != nil {
			serverError(w, err, s.logger)
		}
		return

	} else {
		unsupportedMethodError(w, r.Method, s.logger)
	}
}

const GetTemplateNameField = "name"
const GetTemplateEmailField = "email"

// getTemplateArgs holds the names of the form fields to use in the form.
// They're extracted into the constants above so they are easy to change if need be.
type getTemplateArgs struct {
	Name  string
	Email string
}

type postTemplateArgs struct {
	ID string // The newly created account id.
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
	l.Error().Int("status", http.StatusInternalServerError).Err(err).Send()
}
