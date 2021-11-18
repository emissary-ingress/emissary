//go:generate go run ./types-gen.go

package golist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func GoListPackages(flags []string, pkgnames []string) ([]Package, error) {
	cmdline := []string{"go", "list"}
	cmdline = append(cmdline, flags...)
	cmdline = append(cmdline, "-json", "--")
	cmdline = append(cmdline, pkgnames...)

	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stderr = os.Stderr

	stdoutBytes, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%q: %w", cmdline, err)
	}
	stdoutDecoder := json.NewDecoder(bytes.NewReader(stdoutBytes))
	var ret []Package
	for {
		var pkg Package
		if err := stdoutDecoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		ret = append(ret, pkg)
	}
	return ret, nil
}

func GoListModules(flags []string, modnames []string) ([]Module, error) {
	cmdline := []string{"go", "list"}
	cmdline = append(cmdline, flags...)
	cmdline = append(cmdline, "-m", "-json", "--")
	cmdline = append(cmdline, modnames...)

	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stderr = os.Stderr

	stdoutBytes, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%q: %w", cmdline, err)
	}
	stdoutDecoder := json.NewDecoder(bytes.NewReader(stdoutBytes))
	var ret []Module
	for {
		var mod Module
		if err := stdoutDecoder.Decode(&mod); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if mod.Dir == "" {
			mod.Dir = "vendor/" + mod.Path
		}
		ret = append(ret, mod)
	}
	return ret, nil
}
