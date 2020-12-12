package signup_test

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"roci.dev/diff-server/account"
	"roci.dev/diff-server/serve/signup"
	"roci.dev/diff-server/util/log"
)

func TestGET(t *testing.T) {
	assert := assert.New(t)
	dir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	tmpl := template.Must(template.ParseFiles(signup.TemplateFiles(".")...))
	service := signup.NewService(log.Default(), tmpl, dir)
	m := mux.NewRouter()
	signup.RegisterHandlers(service, m)

	getForm := httptest.NewRequest("GET", signup.Path, nil)
	getFormRecorder := httptest.NewRecorder()
	m.ServeHTTP(getFormRecorder, getForm)
	resp := getFormRecorder.Result()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	body := string(bodyBytes)
	assert.True(strings.Contains(body, fmt.Sprintf(`name="%s"`, signup.GetTemplateNameField)))
	assert.True(strings.Contains(body, fmt.Sprintf(`name="%s"`, signup.GetTemplateEmailField)))
	assert.True(strings.Contains(body, `type="submit"`))
}

func TestPOST(t *testing.T) {
	assert := assert.New(t)
	dir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	defer func() { assert.NoError(os.RemoveAll(dir)) }()

	tmpl := template.Must(template.ParseFiles(signup.TemplateFiles(".")...))
	service := signup.NewService(log.Default(), tmpl, dir)
	m := mux.NewRouter()
	signup.RegisterHandlers(service, m)
	db, err := account.NewDB(dir)
	assert.NoError(err)
	expectedASID := db.HeadValue().NextASID

	postData := url.Values{}
	postData.Set(signup.GetTemplateNameField, "Larry")
	postData.Set(signup.GetTemplateEmailField, "larry@example.com")
	postForm := httptest.NewRequest("POST", signup.Path, bytes.NewBufferString(postData.Encode()))
	postForm.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postFormRecorder := httptest.NewRecorder()
	m.ServeHTTP(postFormRecorder, postForm)
	resp := postFormRecorder.Result()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(err)

	// Ensure repsonse is what we expect.
	assert.Equal(200, resp.StatusCode)
	body := string(bodyBytes)
	assert.True(strings.Contains(body, fmt.Sprintf("ID is %d", expectedASID)))

	// Ensure the account db was updated.
	assert.NoError(db.Reload())
	hv := db.HeadValue()
	assert.Equal(expectedASID+1, hv.NextASID)
	assert.Equal("Larry", hv.Record[expectedASID].Name)
	assert.Equal("larry@example.com", hv.Record[expectedASID].Email)
	assert.NotEqual("", hv.Record[expectedASID].DateCreated)
}
