package environment

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	ambassadorRootEnvVar          = "APPDIR"
	ambassadorConfigBaseDirEnvVar = "AMBASSADOR_CONFIG_BASE_DIR"
	ambassadorClusterIdEnvVar     = "AMBASSADOR_CLUSTER_ID"
)

// EnvironmentSetupEntrypoint replicates the entrypoint.sh environment bootstrapping if the docker entrypoint was changed
func EnvironmentSetupEntrypoint() {
	if os.Getenv(ambassadorClusterIdEnvVar) != "" {
		return
	}
	ambassadorRoot := "/ambassador"
	ambassadorConfigBaseDir := ambassadorRoot
	if os.Getenv(ambassadorRootEnvVar) != "" {
		ambassadorRoot = os.Getenv(ambassadorRootEnvVar)
	}
	if os.Getenv(ambassadorConfigBaseDirEnvVar) != "" {
		ambassadorConfigBaseDir = os.Getenv(ambassadorConfigBaseDirEnvVar)
	}

	// build kubewatch.py command
	cmd := exec.Command("python3", ambassadorRoot+"/kubewatch.py", "--debug")

	// inherit all existing environment variables & inject python's own
	cmd.Env = os.Environ()
	if os.Getenv("PYTHON_EGG_CACHE") == "" {
		cmd.Env = append(cmd.Env, "PYTHON_EGG_CACHE="+ambassadorConfigBaseDir+"/.cache")
	}
	cmd.Env = append(cmd.Env, "PYTHONUNBUFFERED=true")

	// execute and read output
	out, err := cmd.Output()
	if err != nil {
		log.Printf("%s failed with %s\n%s\n", cmd.String(), err, string(out))
		return
	}

	// set the AMBASSADOR_CLUSTER_ID
	os.Setenv(ambassadorClusterIdEnvVar, strings.TrimSpace(string(out)))
	log.Printf("%s=%s", ambassadorClusterIdEnvVar, os.Getenv(ambassadorClusterIdEnvVar))
}
