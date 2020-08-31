package entrypoint

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

func GetAgentService() string {
	return env("AGENT_SERVICE", "")
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

func GetAmbassadorConfigBaseDir() string {
	return env("AMBASSADOR_CONFIG_BASE_DIR", GetAmbassadorRoot())
}

func GetEnvoyDir() string {
	return env("ENVOY_DIR", path.Join(GetAmbassadorConfigBaseDir(), "envoy"))
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

func GetConfigDir() string {
	return env("config_dir", path.Join(GetAmbassadorConfigBaseDir(), "ambassador-config"))
}

func GetSnapshotDir() string {
	return env("snapshot_dir", path.Join(GetAmbassadorConfigBaseDir(), "snapshots"))
}

func GetEnvoyConfigFile() string {
	return env("envoy_config_file", path.Join(GetEnvoyDir(), "envoy.json"))
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
		result = append(result, "-l", "debug")
	} else {
		result = append(result, "-l", "error")
	}
	return result
}

func GetDiagdBindAddress() string {
	return env("AMBASSADOR_DIAGD_BIND_ADDREASS", "")
}

func IsDiagdOnly() bool {
	return envbool("DIAGD_ONLY")
}

func GetDiagdBindPort() string {
	return env("AMBASSADOR_DIAGD_BIND_PORT", "8877")
}

func IsEnvoyAvailable() bool {
	_, err := exec.LookPath("envoy")
	return err == nil
}

func GetDiagdFlags() []string {
	result := []string{"--notices", path.Join(GetAmbassadorConfigBaseDir(), "notices.json")}
	if isDebug("diagd") {
		result = append(result, "--debug")
	}
	diagdBind := GetDiagdBindAddress()
	if diagdBind != "" {
		result = append(result, "--host", diagdBind)
	}
	// XXX: this was not in entrypoint.sh
	result = append(result, "--port", GetDiagdBindPort())
	if IsDiagdOnly() {
		result = append(result, "--no-checks", "--no-envoy")
	} else {
		result = append(result, "--kick", fmt.Sprintf("kill -HUP %d", os.Getpid()))
		// XXX: this was not in entrypoint.sh
		if !IsEnvoyAvailable() {
			result = append(result, "--no-envoy")
		}
	}
	return result
}

func GetDiagdArgs() []string {
	return append([]string{GetSnapshotDir(), GetEnvoyBootstrapFile(), GetEnvoyConfigFile()}, GetDiagdFlags()...)
}

func IsAmbassadorSingleNamespace() bool {
	return envbool("AMBASSADOR_SINGLE_NAMESPACE")
}

func IsEdgeStack() bool {
	_, err := os.Stat("/ambassador/.edge_stack")
	if err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		panic(err)
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
