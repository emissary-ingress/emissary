package dtest

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
)

const scope = "dtest"

func lines(str string) []string {
	var result []string

	for _, l := range strings.Split(str, "\n") {
		l := strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}

	return result
}

func dockerPs(ctx context.Context, args ...string) []string {
	cmd := dexec.CommandContext(ctx, "docker", append([]string{"ps", "-q", "-f", fmt.Sprintf("label=scope=%s", scope)},
		args...)...)
	cmd.DisableLogging = true
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	return lines(string(out))
}

func tag2id(ctx context.Context, tag string) string {
	dlog.Printf(ctx, "Resolving Docker label=%q to a container ID...", tag)
	result := dockerPs(ctx, "-f", fmt.Sprintf("label=%s", tag))
	switch len(result) {
	case 0:
		dlog.Printf(ctx, "no label=%q container", tag)
		return ""
	case 1:
		dlog.Printf(ctx, "label=%q refers to container ID %q", tag, result[0])
		return result[0]
	default:
		panic(fmt.Sprintf("expecting zero or one containers with label scope=%s and label %s", scope, tag))
	}
}

func dockerUp(ctx context.Context, tag string, args ...string) string {
	var id string

	WithNamedMachineLock(ctx, "docker", func(ctx context.Context) {
		dlog.Printf(ctx, "Bringing up Docker container label=%q...", tag)
		id = tag2id(ctx, tag)

		if id == "" {
			runArgs := []string{
				"run",
				"-d",
				"--rm",
				fmt.Sprintf("--label=scope=%s", scope),
				fmt.Sprintf("--label=%s", tag),
				fmt.Sprintf("--name=%s-%s", scope, tag),
			}
			cmd := dexec.CommandContext(ctx, "docker", append(runArgs, args...)...)
			out, err := cmd.Output()
			if err != nil {
				panic(err)
			}
			id = strings.TrimSpace(string(out))[:12]
		}
	})

	return id
}

func dockerKill(ctx context.Context, ids ...string) {
	if len(ids) > 0 {
		cmd := dexec.CommandContext(ctx, "docker", append([]string{"kill"}, ids...)...)
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}
}

func isKubeconfigReady(ctx context.Context) bool {
	id := tag2id(ctx, "k3s")

	if id == "" {
		return false
	}

	cmd := dexec.CommandContext(ctx, "docker", "exec", "-i", id, "sh", "-c",
		"if test -e /etc/rancher/k3s/k3s.yaml; then echo true; else echo false; fi")
	outBytes, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	outString := strings.TrimSpace(string(outBytes))
	ret, err := strconv.ParseBool(outString)
	if err != nil {
		panic(fmt.Errorf("ParseBool: %q: %w", outString, err))
	}
	return ret
}

var requiredResources = []string{
	"bindings",
	"componentstatuses",
	"configmaps",
	"endpoints",
	"events",
	"limitranges",
	"namespaces",
	"nodes",
	"persistentvolumeclaims",
	"persistentvolumes",
	"pods",
	"podtemplates",
	"replicationcontrollers",
	"resourcequotas",
	"secrets",
	"serviceaccounts",
	"services",
	"mutatingwebhookconfigurations.admissionregistration.k8s.io",
	"validatingwebhookconfigurations.admissionregistration.k8s.io",
	"customresourcedefinitions.apiextensions.k8s.io",
	"apiservices.apiregistration.k8s.io",
	"controllerrevisions.apps",
	"daemonsets.apps",
	"deployments.apps",
	"replicasets.apps",
	"statefulsets.apps",
	"tokenreviews.authentication.k8s.io",
	"localsubjectaccessreviews.authorization.k8s.io",
	"selfsubjectaccessreviews.authorization.k8s.io",
	"selfsubjectrulesreviews.authorization.k8s.io",
	"subjectaccessreviews.authorization.k8s.io",
	"horizontalpodautoscalers.autoscaling",
	"cronjobs.batch",
	"jobs.batch",
	"certificatesigningrequests.certificates.k8s.io",
	"leases.coordination.k8s.io",
	"endpointslices.discovery.k8s.io",
	"events.events.k8s.io",
	"ingresses.extensions",
	"helmcharts.helm.cattle.io",
	"addons.k3s.cattle.io",
	"nodes.metrics.k8s.io",
	"pods.metrics.k8s.io",
	"ingresses.networking.k8s.io",
	"networkpolicies.networking.k8s.io",
	"runtimeclasses.node.k8s.io",
	"poddisruptionbudgets.policy",
	"podsecuritypolicies.policy",
	"clusterrolebindings.rbac.authorization.k8s.io",
	"clusterroles.rbac.authorization.k8s.io",
	"rolebindings.rbac.authorization.k8s.io",
	"roles.rbac.authorization.k8s.io",
	"priorityclasses.scheduling.k8s.io",
	"csidrivers.storage.k8s.io",
	"csinodes.storage.k8s.io",
	"storageclasses.storage.k8s.io",
	"volumeattachments.storage.k8s.io",
}

func isK3sReady(ctx context.Context) bool {
	kubeconfig := getKubeconfigPath(ctx)

	if kubeconfig == "" {
		return false
	}

	cmd := dexec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "api-resources", "-o", "name")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	resources := make(map[string]bool)
	for _, line := range strings.Split(string(output), "\n") {
		resources[strings.TrimSpace(line)] = true
	}

	for _, req := range requiredResources {
		_, exists := resources[req]
		if !exists {
			return false
		}
	}

	get := dexec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "get", "namespace", "default")
	err = get.Start()
	if err != nil {
		panic(err)
	}
	return get.Wait() == nil
}

const k3sConfigPath = "/etc/rancher/k3s/k3s.yaml"

// GetKubeconfig returns the kubeconfig contents for the running k3s
// cluster as a string. It will return the empty string if no cluster
// is running.
func GetKubeconfig(ctx context.Context) string {
	if !isKubeconfigReady(ctx) {
		return ""
	}

	id := tag2id(ctx, "k3s")

	if id == "" {
		return ""
	}

	cmd := dexec.CommandContext(ctx, "docker", "exec", "-i", id, "cat", k3sConfigPath)
	kubeconfigBytes, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	kubeconfig := strings.ReplaceAll(string(kubeconfigBytes), "localhost:6443", net.JoinHostPort(dockerIP(), k3sPort))
	return kubeconfig
}

func getKubeconfigPath(ctx context.Context) string {
	id := tag2id(ctx, "k3s")

	if id == "" {
		return ""
	}

	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	kubeconfig := fmt.Sprintf("/tmp/dtest-kubeconfig-%s-%s.yaml", user.Username, id)
	contents := GetKubeconfig(ctx)

	err = ioutil.WriteFile(kubeconfig, []byte(contents), 0644)

	if err != nil {
		panic(err)
	}

	return kubeconfig
}

const dtestRegistry = "DTEST_REGISTRY"
const registryPort = "5000"

// RegistryUp will launch if necessary and return the docker id of a
// container running a docker registry.
func RegistryUp(ctx context.Context) string {
	dlog.Printf(ctx, "Bringing up registry...")
	return dockerUp(ctx, "registry",
		"-p", fmt.Sprintf("%s:6443", k3sPort),
		"-p", fmt.Sprintf("%s:%s", registryPort, registryPort),
		"-e", fmt.Sprintf("REGISTRY_HTTP_ADDR=0.0.0.0:%s", registryPort),
		"registry:2")
}

func dockerIP() string {
	return "localhost"
}

// DockerRegistry returns a docker registry suitable for use in tests.
func DockerRegistry(ctx context.Context) string {
	registry := os.Getenv(dtestRegistry)
	if registry != "" {
		return registry
	}

	RegistryUp(ctx)

	return fmt.Sprintf("%s:%s", dockerIP(), registryPort)
}

const dtestKubeconfig = "DTEST_KUBECONFIG"
const k3sPort = "6443"
const k3sImage = "rancher/k3s:v1.21.1-k3s1"

const k3sMsg = `
kubeconfig does not exist: %s

  Make sure DTEST_KUBECONFIG is either unset or points to a valid kubeconfig file.

`

// Kubeconfig returns a path referencing a kubeconfig file suitable for use in tests.
func Kubeconfig(ctx context.Context) string {
	kubeconfig := os.Getenv(dtestKubeconfig)
	if kubeconfig != "" {
		if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
			fmt.Printf(k3sMsg, kubeconfig)
			os.Exit(1)
		}

		return kubeconfig
	}

	K3sUp(ctx)

	dlog.Printf(ctx, "Polling for k3s to be ready...")
	for ctx.Err() == nil {
		if isK3sReady(ctx) {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
	dlog.Printf(ctx, "k3s is ready!")

	return getKubeconfigPath(ctx)
}

// K3sUp will launch if necessary and return the docker id of a
// container running a k3s cluster.
func K3sUp(ctx context.Context) string {
	regid := RegistryUp(ctx)
	dlog.Printf(ctx, "Bringing up k3s...")
	return dockerUp(ctx, "k3s",
		// `docker run` flags
		"--privileged",
		"--network=container:"+regid,
		"--volume=/dev/mapper:/dev/mapper",
		"--entrypoint=/bin/sh", // for the cgroup hack below
		// Docker image
		k3sImage,
		// https://github.com/k3s-io/k3s/pull/3237
		"-c", `
			if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
			  mkdir -p /sys/fs/cgroup/init
			  busybox xargs -rn1 </sys/fs/cgroup/cgroup.procs >/sys/fs/cgroup/init/cgroup.procs
			  sed -e 's/ / +/g' -e 's/^/+/' </sys/fs/cgroup/cgroup.controllers >/sys/fs/cgroup/cgroup.subtree_control
			fi
			exec /bin/k3s "$@"
		`, "--",
		// `k3s` args
		"server",
		"--node-name=localhost",
		"--no-deploy=traefik",
		"--kube-proxy-arg=conntrack-max-per-core=0")
}

// K3sDown shuts down the k3s cluster.
func K3sDown(ctx context.Context) string {
	id := tag2id(ctx, "k3s")
	if id != "" {
		dockerKill(ctx, id)
	}
	return id
}

// RegistryDown shutsdown the test registry.
func RegistryDown(ctx context.Context) string {
	id := tag2id(ctx, "registry")
	if id != "" {
		dockerKill(ctx, id)
	}
	return id
}
