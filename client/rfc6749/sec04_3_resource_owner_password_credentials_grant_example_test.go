package rfc6749_test

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/datawire/apro/client/rfc6749"
)

func ExampleResourceOwnerPasswordCredentialsClient() {
	client, err := rfc6749.NewResourceOwnerPasswordCredentialsClient(
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
	sessionStore := map[string]*rfc6749.ResourceOwnerPasswordCredentialsClientSessionData{}
	var sessionStoreLock sync.Mutex
	LoadSession := func(r *http.Request) (sessionID string, sessionData *rfc6749.ResourceOwnerPasswordCredentialsClientSessionData) {
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
	SaveSession := func(sessionID string, sessionData *rfc6749.ResourceOwnerPasswordCredentialsClientSessionData) {
		sessionStoreLock.Lock()
		sessionStore[sessionID] = sessionData
		sessionStoreLock.Unlock()
	}

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if _, sessionData := LoadSession(r); sessionData != nil {
				w.Header().Set("Content-Type", "text/html")
				_, _ = io.WriteString(w, `<p>Already logged in. <a href="/dashboard">Return to dashboard.</a></p>`)
				return
			}
			sessionID := randomToken()
			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: sessionID,
			})
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `
				<form action="/login" method="POST">
				  <input name="xsrf_token" type="hidden" value="%s" />
				  <label>Username: <input name="username" type="text" /></label>
				  <label>Password: <input name="password" type="password" /></label>
				  <input type="submit" />
				</form>`, sessionID)

		case http.MethodPost:
			if _, sessionData := LoadSession(r); sessionData != nil {
				w.Header().Set("Content-Type", "text/html")
				_, _ = io.WriteString(w, `<p>Already logged in. <a href="/dashboard">Return to dashboard.</a></p>`)
				return
			}
			cookie, _ := r.Cookie("session")
			if cookie == nil || r.PostFormValue("xsrf_token") != cookie.Value {
				http.Error(w, "XSRF attack detected", http.StatusBadRequest)
				return
			}
			sessionID := cookie.Value

			username := r.PostFormValue("username")
			password := r.PostFormValue("password")
			requiredScopes := rfc6749.Scope{
				"scope-a": struct{}{},
				"scope-B": struct{}{},
			}
			sessionData, err := client.AuthorizationRequest(username, password, requiredScopes)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			SaveSession(sessionID, sessionData)

			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
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
