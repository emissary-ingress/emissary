package rfc6749_test

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/datawire/liboauth2/client/rfc6749"
)

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

var errorResponsePage = template.Must(template.New("error-response-page").Parse(`
<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <title>Error Response: {{.Error}}</title>
  </head>
  <body>
    <dl>

      <dt>error code</dt>
      <dd><tt>{{.Error}}</tt> : {{.Error.Description}}</dd>

{{ if .ErrorDescription | ne "" }}
      <dt>error description</dt>
      <dd>{{.ErrorDescription}}</dd>
{{ end }}

{{ if .ErrorURI | ne nil }}
      <dt>error URI</dt>
      <dd><a href="{{.ErrorURI}}">{{.ErrorURI}}</a></dd>
{{ end }}

    </dl>
  </body>
</html>
`))

func randomToken() string {
	d := make([]byte, 128)
	if _, err := rand.Read(d); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(d)
}

func ExampleAuthorizationCodeClient() {
	client, err := rfc6749.NewAuthorizationCodeClient(
		"example-client",
		mustParseURL("https://authorization-server.example.com/authorization"),
		mustParseURL("https://authorization-server.example.com/token"),
		rfc6749.ClientPasswordHeader("example-client", "example-password"),
	)
	if err != nil {
		log.Fatal(err)
	}

	sessionStore := map[string]*rfc6749.AuthorizationCodeClientSessionData{}

	http.HandleFunc("/.well-known/internal/action", func(w http.ResponseWriter, r *http.Request) {
		sessionID := randomToken()

		requiredScopes := rfc6749.Scope{
			"scope-a": struct{}{},
			"scope-B": struct{}{},
		}
		u, sessionData, err := client.AuthorizationRequest(
			mustParseURL("https://example-client.example.com/redirection"),
			requiredScopes,
			randomToken())
		if err != nil {
			http.Error(w, "could not construct authorization request URI", http.StatusInternalServerError)
			return
		}
		sessionStore[sessionID] = sessionData

		http.SetCookie(w, &http.Cookie{
			Name:  "session",
			Value: sessionID,
		})
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	})

	http.HandleFunc("/.well-known/internal/redirecton", func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sessionData := sessionStore[cookie.Value]
		if sessionData == nil {
			http.Error(w, "unrecognized session ID", http.StatusBadRequest)
			return
		}

		authorizationCode, err := client.ParseAuthorizationResponse(sessionData, r.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tokenResponse, err := client.AccessToken(sessionData, nil, authorizationCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		switch tokenResponse := tokenResponse.(type) {
		case rfc6749.TokenErrorResponse:
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusUnauthorized)
			_ = errorResponsePage.Execute(w, tokenResponse)
			return
		case rfc6749.TokenSuccessResponse:
			// TODO
		}
	})

	log.Println("Listening on :9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
