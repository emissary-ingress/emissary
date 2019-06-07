package rfc6749_test

import (
	"log"
	"net/http"

	"github.com/datawire/liboauth2/client/rfc6749"
)

func ExampleAuthorizationCodeClient() {
	client, err := rfc6749.NewAuthorizationCodeClient(
		"example-client",
		mustParseURL("https://authorization-server.example.com/authorization"),
		mustParseURL("https://authorization-server.example.com/token"),
		rfc6749.ClientPasswordHeader("example-client", "example-password"),
		nil,
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

		tokenResponse, err := client.AccessToken(sessionData, authorizationCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		// TODO
		log.Println(tokenResponse)
	})

	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sessionData := sessionStore[cookie.Value]

		// TODO
		log.Println(sessionData)
	})

	log.Println("Listening on :9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
