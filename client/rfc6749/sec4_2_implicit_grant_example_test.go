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

func randomToken() string {
	d := make([]byte, 128)
	if _, err := rand.Read(d); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(d)
}

func ExampleAuthorizationCodeClient() {
	client, err := rfc6749.NewImplicitClient(
		"example-client",
		mustParseURL("https://authorization-server.example.com/authorization"),
	)
	if err != nil {
		log.Fatal(err)
	}

	sessionStore := map[string]*rfc6749.AuthorizationCodeClientSessionData{}

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		sessionID := randomToken()

		requiredScopes := rfc6749.Scope{
			"scope-a": struct{}{},
			"scope-B": struct{}{},
		}
		u, sessionData, err := client.AuthorizationRequest(
			mustParseURL("https://example-client.example.com/dashboard"),
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
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<script>window.location = "/.well-known/internal/redirection_helper?fragment=" + encodeURIComponent(window.location.hash.substring(1))</script>`)
	})

	http.HandleFunc("/.well-known/internal/redirecton_helper", func(w http.ResponseWriter, r *http.Request) {
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

		fragment := r.URL.Query().Get("fragment")

		token, err := client.ParseAccessTokenResponse(sessionData, fragment)
		// TODO
	})

	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sessionData := sessionStore[cookie.Value]

		// TODO
	})

	log.Println("Listening on :9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
