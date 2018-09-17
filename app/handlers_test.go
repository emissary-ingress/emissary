package app_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/datawire/ambassador-oauth/app"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/logger"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
)

func TestMain(m *testing.M) {
	// Setup
	os.Setenv("AUTH_AUDIENCE", "friends")
	os.Setenv("AUTH_DOMAIN", "test.url")
	os.Setenv("AUTH_CALLBACK_URL", "test.url/callback")
	os.Setenv("AUTH_CLIENT_ID", "123")

	ok := m.Run()
	// Teardown
	// ...

	// Exit
	os.Exit(ok)
}

func TestAuthorizeHandler(t *testing.T) {
	// Config
	config := config.New()
	logger := logger.New(config)
	secret := secret.New(config, logger)
	ctrl := &app.Controller{Logger: logger, Config: config}

	ctrl.Rules.Store(make([]app.Rule, 1))

	// Handler
	hdr := app.Handler{
		Config: config,
		Logger: logger,
		Ctrl:   ctrl,
		Secret: secret,
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(hdr.Authorize)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusSeeOther)
	}

	// Check the response body size is what we expect.
	if actual := len(rr.Body.String()); actual != 709 {
		t.Errorf("handler returned unexpected body: got %v want %v in %s", len(rr.Body.String()), 709, rr.Body.String())
	}
}
