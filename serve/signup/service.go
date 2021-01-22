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
		{Name: PostFailureTemplateName, Content: PostFailureTemplate},
		{Name: PostSuccessTemplateName, Content: PostSuccessTemplate},
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
	if r.Method == "GET" {
		if err := s.tmpl.ExecuteTemplate(w, GetTemplateName, getTemplateArgs{GetTemplateNameField, GetTemplateEmailField}); err != nil {
			serverError(w, err, s.logger)
		}
		return

	} else if r.Method == "POST" {
		name := r.FormValue(GetTemplateNameField)
		email := r.FormValue(GetTemplateEmailField)

		// The lightest of all possible lightweight form validations.
		validationFailures := []string{}
		if name == "" {
			validationFailures = append(validationFailures, "Please enter a Name (either your personal name or an entity, eg your company).")
		}
		if strings.Index(email, "@") == -1 {
			validationFailures = append(validationFailures, "Please enter a valid Email Address so we can contact you in the event of problems.")
		}
		if len(validationFailures) > 0 {
			templateArgs := postFailureTemplateArgs{Reasons: validationFailures}
			if err := s.tmpl.ExecuteTemplate(w, PostFailureTemplateName, templateArgs); err != nil {
				serverError(w, err, s.logger)
			}
			return
		}

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
			// TODO: retry if head was changed from under us.
			serverError(w, err, s.logger)
			return
		}
		templateArgs := postSuccessTemplateArgs{ID: fmt.Sprintf("%d", id)}
		if err := s.tmpl.ExecuteTemplate(w, PostSuccessTemplateName, templateArgs); err != nil {
			serverError(w, err, s.logger)
		}
		s.logger.Info().Msgf("Created auto-signup account: %#v", accounts.Record[id])
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

type postSuccessTemplateArgs struct {
	ID string // The newly created account id.
}

type postFailureTemplateArgs struct {
	Reasons []string
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
