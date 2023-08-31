package metriton

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// StaticInstallID is returns an install-ID-getter that always returns
// a fixed install ID.
func StaticInstallID(id string) func(*Reporter) (string, error) {
	return func(*Reporter) (string, error) {
		return id, nil
	}
}

// This is the same as os.UserConfigDir() on GOOS=linux.  We Use this
// instead of os.UserConfigDir() because on we want the GOOS=linux
// behavior on macOS
func userConfigDir() (string, error) {
	var dir string
	dir = os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
		}
		dir += "/.config"
	}
	return dir, nil
}

// InstallIDFromFilesystem is an install-ID-getter that tracks the
// install ID in the filesystem.
func InstallIDFromFilesystem(reporter *Reporter) (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}

	idFilename := filepath.Join(dir, reporter.Application, "id")
	if idBytes, err := ioutil.ReadFile(idFilename); err == nil {
		reporter.BaseMetadata["new_install"] = false
		return string(idBytes), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	id := uuid.New().String()
	reporter.BaseMetadata["new_install"] = true

	if err := os.MkdirAll(filepath.Dir(idFilename), 0755); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(idFilename, []byte(id), 0644); err != nil {
		return "", err
	}
	return id, nil
}
