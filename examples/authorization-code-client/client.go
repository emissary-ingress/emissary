package main

import (
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

func main() {
	client, err := rfc6749.NewAuthorizationCodeClient(
		"example-client",
		mustParseURL("https://authorization-server.example.com/authorization"),
		mustParseURL("https://authorization-server.example.com/token"),
		rfc6749.ClientPasswordHeader("example-client", "example-password"),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
		u, err := client.AuthorizationRequest(mustParseURL("https://example-client.example.com/redirection"), rfc6749.Scope{
			"scope-a": struct{}{},
			"scope-B": struct{}{},
		}, "mystate")
		if err != nil {
			http.Error(w, "could not construct authorization request URI", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	})
	http.HandleFunc("/redirecton", func(w http.ResponseWriter, r *http.Request) {
		authorizationResponse, err := client.ParseAuthorizationResponse(r.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if authorizationResponse.GetState() != "mystate" {
			http.Error(w, "invalid state (likely XSRF attempt)", http.StatusBadRequest)
		}
		switch authorizationResponse := authorizationResponse.(type) {
		case rfc6749.AuthorizationCodeAuthorizationErrorResponse:
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusUnauthorized)
			errorResponsePage.Execute(w, authorizationResponse)
			return
		case rfc6749.AuthorizationCodeAuthorizationSuccessResponse:
			tokenResponse, err := client.AccessToken(nil, authorizationResponse.Code, mustParseURL("https://example-client.example.com/redirection"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
			switch tokenResponse := tokenResponse.(type) {
			case rfc6749.TokenErrorResponse:
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusUnauthorized)
				errorResponsePage.Execute(w, tokenResponse)
				return
			case rfc6749.TokenSuccessResponse:
				// TODO
			}
		}
	})

	log.Println("Listening on :9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
