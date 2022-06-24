package entrypoint

import (
	"context"
	"encoding/json"
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

func ensureDir(dirname string) error {
	err := os.MkdirAll(dirname, 0700)
	if err != nil && os.IsExist(err) {
		err = nil
	}
	return err
}

func cidsForLabel(ctx context.Context, label string) ([]string, error) {
	bs, err := dexec.CommandContext(ctx, "docker", "ps", "-q", "-f", "label="+label).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(bs)), nil
}

func subcommand(ctx context.Context, command string, args ...string) *dexec.Cmd {
	cmd := dexec.CommandContext(ctx, command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
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
