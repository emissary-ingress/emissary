package main

import (
	"log"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func main() {
	log.Println("Starting Envoy with CAP_NET_BIND_SERVICE capability")

	header := unix.CapUserHeader{unix.LINUX_CAPABILITY_VERSION_3, int32(os.Getpid())}
	data := unix.CapUserData{}
	if err := unix.Capget(&header, &data); err != nil {
		log.Fatal(err)
	}

	data.Inheritable = (1 << unix.CAP_NET_BIND_SERVICE)

	if err := unix.Capset(&header, &data); err != nil {
		log.Fatal(err)
	}

	log.Println("Succeeded in setting capabilities")

	if err := syscall.Exec("/usr/local/bin/envoy", os.Args, os.Environ()); err != nil {
		log.Fatal(err)
	}
}
