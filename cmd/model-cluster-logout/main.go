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
		<fieldset><legend>SSR</legend>
			{{ if eq (len .RealmCookies) 0 }}
				<p>Not logged in to any realms.</p>
			{{ else }}
				<ul>{{ range .RealmCookies }}
					<li>
						<form method="POST" action="/.ambassador/oauth2/logout" target="_blank">
							<input type="hidden" name="realm" value="{{ trimprefix .Name "ambassador_xsrf." }}" />
							<input type="hidden" name="_xsrf" value="{{ .Value }}" />
							<input type="submit" value="log out of realm {{ trimprefix .Name "ambassador_xsrf." }}" />
						</form>
					</li>
				{{ end }}</ul>
			{{ end }}
		</fieldset>
		<fieldset><legend>JS</legend>
			{{ .JSApp }}
		</fieldset>
	</body>
</html>
`))

const jsApp = `<div id="app">
	<ul>
		<li v-for="(val, key) in realmCookies">
			<form method="POST" action="/.ambassador/oauth2/logout" target="_blank">
				<input type="hidden" name="realm" v-bind:value="key.slice('ambassador_xsrf.'.length)" />
				<input type="hidden" name="_xsrf" v-bind:value="val" />
				<input type="submit" v-bind:value="'log out of realm '+key.slice('ambassador_xsrf.'.length)" />
			</form>
		</li>
	</ul>
</div>
<script type="module">
	import Vue from 'https://cdn.jsdelivr.net/npm/vue/dist/vue.esm.browser.js';

	function getCookies() {
		let map = {};
		let list = decodeURIComponent(document.cookie).split(';');
		for (let i = 0; i < list.length; i++) {
			let cookie = list[i].trimStart();
			let eq = cookie.indexOf('=');
			let key = cookie.slice(0, eq);
			let val = cookie.slice(eq+1);
			map[key] = val;
		}
		return map;
	}

	new Vue({
		el: '#app',
		data: function() {
			return {
				"cookies": getCookies(),
			};
		},
		computed: {
			"realmCookies": function() {
				let ret = {};
				for (let key in this.cookies) {
					if (key.indexOf("ambassador_xsrf.") == 0) {
						ret[key] = this.cookies[key];
					}
				}
				return ret;
			},
		},
	});
</script>
`

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var realmCookies []*http.Cookie
	for _, cookie := range r.Cookies() {
		if strings.HasPrefix(cookie.Name, "ambassador_xsrf.") {
			realmCookies = append(realmCookies, cookie)
		}
	}
	sort.Slice(realmCookies, func(i, j int) bool {
		return realmCookies[i].Name < realmCookies[j].Name
	})
	w.Header().Set("Content-Type", "text/html; text/html; charset=utf-8")
	tmpl.Execute(w, map[string]interface{}{
		"RealmCookies": realmCookies,
		"JSApp":        jsApp,
	})
}

func main() {
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(ServeHTTP)))
}
