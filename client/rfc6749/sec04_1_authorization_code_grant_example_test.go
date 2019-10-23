package rfc6749_test

import (
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/datawire/apro/client/rfc6749"
)

func ExampleAuthorizationCodeClient() {
	client, err := rfc6749.NewAuthorizationCodeClient(
		"example-client",
		mustParseURL("https://authorization-server.example.com/authorization"),
		mustParseURL("https://authorization-server.example.com/token"),
		rfc6749.ClientPasswordHeader("example-client", "example-password"),
		http.DefaultClient,
	)
	if err != nil {
		log.Fatal(err)
	}

	// This is a toy in-memory store for demonstration purposes.  Because it's in-memory and
	// stores pointers, it isn't actually nescessary to update the store whenever the session
	// data changes.  However, save-on-change is implemented in this example in order to
	// demonstrate how to save it for external data stores.
	sessionStore := map[string]*rfc6749.AuthorizationCodeClientSessionData{}
	var sessionStoreLock sync.Mutex
	LoadSession := func(r *http.Request) (sessionID string, sessionData *rfc6749.AuthorizationCodeClientSessionData) {
		cookie, _ := r.Cookie("session")
		if cookie == nil {
			return "", nil
		}
		sessionID = cookie.Value
		sessionStoreLock.Lock()
		sessionData = sessionStore[sessionID]
		sessionStoreLock.Unlock()
		return sessionID, sessionData
	}
	SaveSession := func(sessionID string, sessionData *rfc6749.AuthorizationCodeClientSessionData) {
		sessionStoreLock.Lock()
		sessionStore[sessionID] = sessionData
		sessionStoreLock.Unlock()
	}

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if _, sessionData := LoadSession(r); sessionData != nil {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, `<p>Already logged in. <a href="/dashboard">Return to dashboard.</a></p>`)
			return
		}

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
		SaveSession(sessionID, sessionData)

		http.SetCookie(w, &http.Cookie{
			Name:  "session",
			Value: sessionID,
		})
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	})

	http.HandleFunc("/.well-known/internal/redirection", func(w http.ResponseWriter, r *http.Request) {
		sessionID, sessionData := LoadSession(r)
		if sessionData == nil {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, `<p><a href="/login">Click to log in</a></p>`)
			return
		}
		defer func() {
			if sessionData.IsDirty() {
				SaveSession(sessionID, sessionData)
			}
		}()

		authorizationCode, err := client.ParseAuthorizationResponse(sessionData, r.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = client.AccessToken(sessionData, authorizationCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// TODO
		log.Println(sessionData)
	})

	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		sessionID, sessionData := LoadSession(r)
		if sessionData == nil {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, `<p><a href="/login">Click to log in</a></p>`)
			return
		}
		defer func() {
			if sessionData.IsDirty() {
				SaveSession(sessionID, sessionData)
			}
		}()

		// TODO
		log.Println(sessionData)
	})
}
