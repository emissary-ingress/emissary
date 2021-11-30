//go:build !linux
// +build !linux

package main

import (
	"errors"
)

func capset() error {
	return errors.New("setcap is only implemented on GOOS=linux")
}
