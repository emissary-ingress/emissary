package main

import (
	"log"
	"os"
	"syscall"
)

func main() {
	log.Println("Starting Envoy with CAP_NET_BIND_SERVICE capability")

	if err := capset(); err != nil {
		log.Fatal(err)
	}

	log.Println("Succeeded in setting capabilities")

	if err := syscall.Exec("/usr/local/bin/envoy", os.Args, os.Environ()); err != nil {
		log.Fatal(err)
	}
}
