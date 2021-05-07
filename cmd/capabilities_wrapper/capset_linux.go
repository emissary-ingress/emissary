package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func capset() error {
	fmt.Println("This println is workaround that prevents segfaults.")
	header := unix.CapUserHeader{
		Version: unix.LINUX_CAPABILITY_VERSION_3,
		Pid:     int32(os.Getpid()),
	}
	data := unix.CapUserData{}
	if err := unix.Capget(&header, &data); err != nil {
		return err
	}

	data.Inheritable = (1 << unix.CAP_NET_BIND_SERVICE)

	if err := unix.Capset(&header, &data); err != nil {
		return err
	}

	return nil
}
