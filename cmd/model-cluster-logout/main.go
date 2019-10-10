package main

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"text/template"
)

var tmpl = template.Must(template.
	New("logout.html").
	Funcs(template.FuncMap{
		"trimprefix": strings.TrimPrefix,
	}).
	Parse(`<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <title>Demo logout microservice</title>
  </head>
  <body>
    {{ if eq (len .RealmCookies) 0 }}
      <p>Not logged in to any realms.</p>
    {{ else }}
      <ul>{{ range .RealmCookies }}
        <li>
          <form method="POST" action="/.ambassador/oauth2/logout">
            <input type="hidden" name="realm" value="{{ trimprefix .Name "ambassador_session." }}" />
            <input type="hidden" name="_xsrf" value="{{ .Value }}" />
            <input type="submit" value="log out of realm {{ trimprefix .Name "ambassador_session." }}" />
          </form>
        </li>
      {{ end }}</ul>
    {{ end }}
  </body>
</html>
`))

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var realmCookies []*http.Cookie
	for _, cookie := range r.Cookies() {
		if strings.HasPrefix(cookie.Name, "ambassador_session.") {
			realmCookies = append(realmCookies, cookie)
		}
	}
	sort.Slice(realmCookies, func(i, j int) bool {
		return realmCookies[i].Name < realmCookies[j].Name
	})
	w.Header().Set("Content-Type", "text/html; text/html; charset=utf-8")
	tmpl.Execute(w, map[string]interface{}{
		"RealmCookies": realmCookies,
	})
}

func main() {
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(ServeHTTP)))
}
