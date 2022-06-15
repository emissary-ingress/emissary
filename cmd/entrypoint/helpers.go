package entrypoint

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/datawire/dlib/dexec"
	amb "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
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

// Should we pay attention to a given AmbassadorID set?
//
// XXX Yes, amb.AmbassadorID is a singular name for a plural type. Sigh.
func include(id amb.AmbassadorID) bool {
	// We always pay attention to the "_automatic_" ID -- it gives us a
	// to easily always include certain configuration resources for Edge
	// Stack.
	if len(id) == 1 && id[0] == "_automatic_" {
		return true
	}

	// It's not "_automatic_", so we have to actually do the work. Grab
	// our AmbassadorID...
	me := GetAmbassadorId()

	// ...force an empty AmbassadorID to "default", per the documentation...
	if len(id) == 0 {
		id = amb.AmbassadorID{"default"}
	}

	// ...and then see if our AmbassadorID is in the list.
	for _, name := range id {
		if me == name {
			return true
		}
	}

	return false
}
