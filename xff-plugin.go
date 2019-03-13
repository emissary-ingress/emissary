package main

import (
	"log"
	"net/http"
	"strings"
)

func PluginMain(w http.ResponseWriter, r *http.Request) {

	var xffIn = r.Header.Get("X-Forwarded-For")

	log.Printf("Incoming XFF: %s\n", xffIn)

	stringSlice := strings.Split(xffIn, ",")

	clientIP := stringSlice[0]

	log.Printf("Client IP: %s\n", clientIP)

	var xffOut = clientIP + ",172.69.138.24"

	log.Printf("Outgoing XFF: %s\n", xffOut)

	w.Header().Set("X-Forwarded-For", xffOut)
	w.Header().Set("X-Envoy-External-Address", "172.69.138.24")
	w.WriteHeader(http.StatusOK)
}
