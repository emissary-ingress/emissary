package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/datawire/dlib/dexec"
)

func envbool(name string) bool {
	return os.Getenv(name) != ""
}

func env(name, defaultValue string) string {
	value := os.Getenv(name)
	if value != "" {
		return value
	} else {
		return defaultValue
	}
}

func ensureDir(dirname string) {
	if !fileExists(dirname) {
		err := os.MkdirAll(dirname, 0700)
		if err != nil {
			panic(err)
		}
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func sh(ctx context.Context, command string, args ...string) string {
	cmd := dexec.CommandContext(ctx, command, args...)
	out, err := cmd.CombinedOutput()
	panicExecError(fmt.Sprintf("error executing command %s %v", command, args), err)
	return string(out)
}

func cidsForLabel(ctx context.Context, label string) []string {
	return strings.Fields(sh(ctx, "docker", "ps", "-q", "-f", fmt.Sprintf("label=%s", label)))
}

func subcommand(ctx context.Context, command string, args ...string) *dexec.Cmd {
	cmd := dexec.CommandContext(ctx, command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func panicExecError(prefix string, err error) {
	if err == nil {
		return
	}

	msg := fmt.Sprintf("%s: %v", prefix, err)
	if exerr, ok := err.(*dexec.ExitError); ok {
		if exerr.Success() {
			return
		}
		msg = fmt.Sprintf("%s\n%s", msg, string(exerr.Stderr))
	}
	panic(msg)
}

func logExecError(prefix string, err error) {
	if err == nil {
		return
	}

	msg := fmt.Sprintf("%s: %v", prefix, err)
	if exerr, ok := err.(*dexec.ExitError); ok {
		if exerr.Success() {
			return
		}
		log.Printf("%s\n%s", msg, string(exerr.Stderr))
	} else {
		// This means we didn't even start the subcommand, so this is a programming error, not a
		// runtime error and we want to panic in this case.
		panic(msg)
	}
}

func convert(in interface{}, out interface{}) error {
	if out == nil {
		return nil
	}

	jsonBytes, err := json.Marshal(in)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonBytes, out)
	if err != nil {
		return err
	}

	return nil
}
