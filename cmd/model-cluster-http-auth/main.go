package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
)

func main() {
	socket, err := net.Listen("tcp", ":4000") // #nosec G102
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/frobnitz/", http.StripPrefix("/frobnitz", &AuthService{}))

	log.Print("starting...")
	log.Fatal(http.Serve(socket, nil))
}

type AuthService struct{}

func (s *AuthService) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Println("ACCESS",
		req.Method,
		req.Host,
		req.URL,
	)
	switch req.URL.Path {
	case "/external-http/headers":
		log.Print("=> ALLOW")
		inputHeaders := make([]string, 0, len(req.Header))
		for k := range req.Header {
			inputHeaders = append(inputHeaders, k)
		}
		sort.Strings(inputHeaders)
		res.Header().Set("X-Input-Headers", strings.Join(inputHeaders, ","))
		res.Header().Set("X-Allowed-Output-Header", "baz")
		res.Header().Set("X-Disallowed-Output-Header", "qux")
		res.WriteHeader(http.StatusOK)
	case "/external-http/redirect":
		log.Print("=> DENY")
		res.Header().Set("Location", "https://example.com/")
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusFound)
		io.WriteString(res, `{"msg": "redirected"}`)
	default:
		log.Print("=> DENY")
		res.Header().Set("X-Allowed-Output-Header", "baz")
		res.Header().Set("X-Disallowed-Output-Header", "qux")
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusNotFound)
		io.WriteString(res, `{"msg": "intercepted"}`)
	}
}
