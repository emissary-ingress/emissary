package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"

	"speclib2/rfc6749client"
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
	client, err := rfc6749client.NewAuthorizationCodeClient(
		"example-client",
		mustParseURL("https://authorization-server.example.com/authorization"),
		mustParseURL("https://authorization-server.example.com/token"),
		rfc6749client.ClientPassword("example-client", "example-password"),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
		client.AuthorizationRequest(w, r, mustParseURL("https://example-client.example.com/redirection"), rfc6749client.Scope{
			"scope-a": struct{}{},
			"scope-B": struct{}{},
		}, "mystate")
	})
	http.HandleFunc("/redirecton", func(w http.ResponseWriter, r *http.Request) {
		authorizationResponse, err := client.ParseAuthorizationResponse(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if authorizationResponse.GetState() != "mystate" {
			http.Error(w, "invalid state (likely XSRF attempt)", http.StatusBadRequest)
		}
		switch authorizationResponse := authorizationResponse.(type) {
		case rfc6749client.AuthorizationCodeAuthorizationErrorResponse:
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusUnauthorized)
			errorResponsePage.Execute(w, authorizationResponse)
			return
		case rfc6749client.AuthorizationCodeAuthorizationSuccessResponse:
			tokenResponse, err := client.AccessToken(nil, authorizationResponse.Code, mustParseURL("https://example-client.example.com/redirection"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
			switch tokenResponse.(type) {
			case rfc6749client.TokenErrorResponse:
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusUnauthorized)
				errorResponsePage.Execute(w, authorizationResponse)
				return
			case rfc6749client.TokenSuccessResponse:
				// TODO
			}
		}
	})

	log.Println("Listening on :9000...")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
