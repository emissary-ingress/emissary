package main

import (
	"log"
	"net"
	"net/http"
)

func main() {
	socket, err := net.Listen("tcp", ":3000") // #nosec G102
	if err != nil {
		log.Fatal(err)
	}

	log.Print("starting...")
	log.Fatal(http.Serve(socket, &AuthService{}))
}

type AuthService struct{}

func (s *AuthService) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
}
