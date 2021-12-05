package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/datawire/dlib/dlog"
)

func GetAgentService() string {
	// Using an agent service is no longer supported, so return empty.
	// For good measure, we also set AGENT_SERVICE to empty in the entrypoint.
	return ""
}

func GetAmbassadorId() string {
	id := os.Getenv("AMBASSADOR_ID")
	if id != "" {
		return id
	}
	svc := GetAgentService()
	if svc != "" {
		return fmt.Sprintf("intercept-%s", svc)
	}
	return "default"
}

func GetAmbassadorNamespace() string {
	return env("AMBASSADOR_NAMESPACE", "default")
}

func GetAmbassadorFieldSelector() string {
	return env("AMBASSADOR_FIELD_SELECTOR", "")
}

func GetAmbassadorLabelSelector() string {
	return env("AMBASSADOR_LABEL_SELECTOR", "")
}

func GetAmbassadorRoot() string {
	return env("ambassador_root", "/ambassador")
}

func GetHomeDir() string {
	return env("HOME", "/tmp/ambassador")
}

func GetAmbassadorConfigBaseDir() string {
	return env("AMBASSADOR_CONFIG_BASE_DIR", GetAmbassadorRoot())
}

func GetEnvoyDir() string {
	return env("ENVOY_DIR", path.Join(GetAmbassadorConfigBaseDir(), "envoy"))
}

func GetEnvoyConcurrency() string {
	return env("ENVOY_CONCURRENCY", "")
}

func GetEnvoyBootstrapFile() string {
	return env("ENVOY_BOOTSTRAP_FILE", path.Join(GetAmbassadorConfigBaseDir(), "bootstrap-ads.json"))
}

func GetEnvoyBaseId() string {
	return env("AMBASSADOR_ENVOY_BASE_ID", "0")
}

func GetAppDir() string {
	return env("APPDIR", GetAmbassadorRoot())
}

// GetConfigDir returns the path to the directory we should check for
// filesystem config.
func GetConfigDir(demoMode bool) string {
	// XXX There was no way to override the config dir via the environment in the old
	// entrypoint.sh.
	configDir := env("AMBASSADOR_CONFIG_DIR", path.Join(GetAmbassadorConfigBaseDir(), "ambassador-config"))

	if demoMode {
		// There is _intentionally_ no way to override the demo-mode config directory,
		// and it is _intentionally_ based on the root directory rather than on
		// AMBASSADOR_CONFIG_BASE_DIR: it's baked into a specific location during
		// the build process.
		configDir = path.Join(GetAmbassadorRoot(), "ambassador-demo-config")
	}

	return configDir
}

// ConfigIsPresent checks to see if any configuration is actually present
// in the given configdir.
func ConfigIsPresent(ctx context.Context, configDir string) bool {
	// Is there anything in this directory?
	foundAny := false

	_ = filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		// If we're handed an error coming in, something has gone wrong and we _must_
		// return the error to avoid a panic. (The most likely error, admittedly, is
		// simply that the toplevel directory doesn't exist.)
		if err != nil {
			// Log it, but if it isn't an os.ErrNoExist().
			if !os.IsNotExist(err) {
				dlog.Errorf(ctx, "Error scanning config file %s: %s", filepath.Join(configDir, path), err)
			}

			return err
		}

		if (info.Mode() & os.ModeType) == 0 {
			// This is a regular file, so we can consider this valid config.
			foundAny = true

			// Return an error in order to short-circuit the rest of the walk.
			// This is kind of an abuse, honestly, but then we also don't want
			// to spend a long time walking crap if someone sets the environment
			// variable incorrectly -- and if we run into an actual error walking
			// the config dir, see the comment above.
			return errors.New("not really an errore")
		}

		return nil
	})

	// Done. We don't care what the walk actually returned, we only care
	// about foundAny.
	return foundAny
}

func GetSnapshotDir() string {
	return env("snapshot_dir", path.Join(GetAmbassadorConfigBaseDir(), "snapshots"))
}

func GetEnvoyConfigFile() string {
	return env("envoy_config_file", path.Join(GetEnvoyDir(), "envoy.json"))
}

func GetEnvoyAPIVersion() string {
	return env("AMBASSADOR_ENVOY_API_VERSION", "V3")
}

func GetAmbassadorDebug() string {
	return env("AMBASSADOR_DEBUG", "")
}

func isDebug(name string) bool {
	return strings.Contains(GetAmbassadorDebug(), name)
}

func GetEnvoyFlags() []string {
	result := []string{"-c", GetEnvoyBootstrapFile(), "--base-id", GetEnvoyBaseId()}
	svc := GetAgentService()
	if svc != "" {
		result = append(result, "--drain-time-s", "1")
	} else {
		result = append(result, "--drain-time-s", env("AMBASSADOR_DRAIN_TIME", "600"))
	}
	if isDebug("envoy") {
		result = append(result, "-l", "trace")
	} else {
		result = append(result, "-l", "error")
	}
	concurrency := GetEnvoyConcurrency()
	if concurrency != "" {
		result = append(result, "--concurrency", concurrency)
	}
	envoyAPIVersion := GetEnvoyAPIVersion()
	if strings.ToUpper(envoyAPIVersion) == "V3" {
		result = append(result, "--bootstrap-version", "3")
	} else {
		result = append(result, "--bootstrap-version", "2")
	}
	return result
}

func GetDiagdBindAddress() string {
	return env("AMBASSADOR_DIAGD_BIND_ADDREASS", "")
}

func IsDiagdOnly() bool {
	return envbool("DIAGD_ONLY")
}

// ForceEndpoints reflects AMBASSADOR_FORCE_ENDPOINTS, to determine whether
// we're forcing endpoint watching or (the default) not.
func ForceEndpoints() bool {
	return envbool("AMBASSADOR_FORCE_ENDPOINTS")
}

func GetDiagdBindPort() string {
	return env("AMBASSADOR_DIAGD_BIND_PORT", "8004")
}

func IsEnvoyAvailable() bool {
	_, err := exec.LookPath("envoy")
	return err == nil
}

func GetDiagdFlags(ctx context.Context, demoMode bool) []string {
	result := []string{"--notices", path.Join(GetAmbassadorConfigBaseDir(), "notices.json")}

	if isDebug("diagd") {
		result = append(result, "--debug")
	}

	diagdBind := GetDiagdBindAddress()
	if diagdBind != "" {
		result = append(result, "--host", diagdBind)
	}

	// XXX: this was not in the old entrypoint.sh
	result = append(result, "--port", GetDiagdBindPort())

	cdir := GetConfigDir(demoMode)

	if (cdir != "") && ConfigIsPresent(ctx, cdir) {
		result = append(result, "--config-path", cdir)
	}

	if IsDiagdOnly() {
		result = append(result, "--no-checks", "--no-envoy")
	} else {
		result = append(result, "--kick", fmt.Sprintf("kill -HUP %d", os.Getpid()))
		// XXX: this was not in the old entrypoint.sh
		if !IsEnvoyAvailable() {
			result = append(result, "--no-envoy")
		}
	}

	return result
}

func GetDiagdArgs(ctx context.Context, demoMode bool) []string {
	return append(
		[]string{
			GetSnapshotDir(),
			GetEnvoyBootstrapFile(),
			GetEnvoyConfigFile(),
		},
		GetDiagdFlags(ctx, demoMode)...,
	)
}

func IsAmbassadorSingleNamespace() bool {
	return envbool("AMBASSADOR_SINGLE_NAMESPACE")
}

func IsEdgeStack() (bool, error) {
	if envbool("EDGE_STACK") {
		return true, nil
	}
	_, err := os.Stat("/ambassador/.edge_stack")
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func GetLicenseSecretName() string {
	return env("AMBASSADOR_AES_SECRET_NAME", "ambassador-edge-stack")
}

func GetLicenseSecretNamespace() string {
	return env("AMBASSADOR_AES_SECRET_NAMESPACE", GetAmbassadorNamespace())
}

func GetEventHost() string {
	return env("DEV_AMBASSADOR_EVENT_HOST", fmt.Sprintf("http://localhost:%s", GetDiagdBindPort()))
}

func GetEventPath() string {
	return env("DEV_AMBASSADOR_EVENT_PATH", fmt.Sprintf("_internal/v0"))
}

func GetSidecarHost() string {
	return env("DEV_AMBASSADOR_SIDECAR_HOST", "http://localhost:8500")
}

func GetSidecarPath() string {
	return env("DEV_AMBASSADOR_SIDECAR_PATH", "_internal/v0")
}

func GetEventUrl() string {
	return fmt.Sprintf("%s/%s/watt", GetEventHost(), GetEventPath())
}

func GetSidecarUrl() string {
	return fmt.Sprintf("%s/%s/watt", GetSidecarHost(), GetSidecarPath())
}

func IsKnativeEnabled() bool {
	return strings.ToLower(env("AMBASSADOR_KNATIVE_SUPPORT", "")) == "true"
}
