package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/api/agent"
	"github.com/emissary-ingress/emissary/v3/pkg/dtest"
	"github.com/emissary-ingress/emissary/v3/pkg/k8s"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/kubeapply"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// This test is supposed to be a very lightweight end to end test.
// We're essentially testing that the k8s yaml configuration allows the agent to report on all the
// things the cloud app needs. We do this with dtest, which spins up a k3ds cluster by default, or
// you can point it at your own cluster by running `go test` with env var `DTEST_KUBECONFIG=$HOME/.kube/config`
// More complicated business logic tests live in ambassador.git/pkg/agent
func TestAgentE2E(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)
	kubeconfig := dtest.KubeVersionConfig(ctx, dtest.Kube22)
	cli, err := kates.NewClient(kates.ClientConfig{Kubeconfig: kubeconfig})
	require.NoError(t, err)
	// applies all k8s yaml to dtest cluter
	// ambassador, ambassador-agent, rbac, crds, and a fake agentcom that implements the grpc
	// server for the agent
	setup(t, ctx, kubeconfig, cli)

	// eh lets make sure the agent came up
	time.Sleep(time.Second * 3)

	defer deleteArgoResources(t, ctx, kubeconfig)
	hasArgo := false
	reportSnapshot, ambSnapshot := getAgentComSnapshots(t, ctx, kubeconfig, cli, hasArgo)

	// Do actual assertions here. kind of lazy way to retry, but it should work
	assert.NotEmpty(t, reportSnapshot.Identity.ClusterId)
	assert.NotEmpty(t, reportSnapshot.Identity.Hostname)
	assert.NotEmpty(t, reportSnapshot.RawSnapshot)
	assert.NotEmpty(t, reportSnapshot.ApiVersion)
	assert.NotEmpty(t, reportSnapshot.SnapshotTs)
	assert.Equal(t, reportSnapshot.ApiVersion, snapshotTypes.ApiVersion)

	assert.NotEmpty(t, ambSnapshot.Kubernetes)

	// just make sure the stuff we really need for the service catalog is in there
	assert.NotEmpty(t, ambSnapshot.Kubernetes.Services, "No services in snapshot")
	assert.NotEmpty(t, ambSnapshot.Kubernetes.Mappings, "No mappings in snapshot")

	// pods not being empty basically ensures that the rbac in the yaml is correct
	assert.NotEmpty(t, ambSnapshot.Kubernetes.Pods, "No pods found in snapshot")
	assert.Empty(t, ambSnapshot.Kubernetes.ArgoRollouts, "rollouts found in snapshot")
	assert.Empty(t, ambSnapshot.Kubernetes.ArgoApplications, "applications found in snapshot")

	applyArgoResources(t, ctx, kubeconfig, cli)
	hasArgo = true
	reportSnapshot, ambSnapshot = getAgentComSnapshots(t, ctx, kubeconfig, cli, hasArgo)
	assert.NotEmpty(t, ambSnapshot.Kubernetes.ArgoRollouts, "No argo rollouts found in snapshot")
	assert.NotEmpty(t, ambSnapshot.Kubernetes.ArgoApplications, "No argo applications found in snapshot")
}

func getAgentComSnapshots(t *testing.T, ctx context.Context, kubeconfig string, cli *kates.Client, waitArgo bool) (*agent.Snapshot, *snapshotTypes.Snapshot) {
	found := false
	reportSnapshot := &agent.Snapshot{}
	ambSnapshot := &snapshotTypes.Snapshot{}

	// now we're going to go copy the snapshot.json file from our fake agentcom
	// when the agentcom gets a snapshot from the agent, it'll store it at /tmp/snapshot.json
	// we do this in a loop because it might take ambassador and the agent a sec to get into the
	// state we're asserting. this is okay, this test is just to make sure that the agent RBAC
	// is correct and that the agent can talk to the ambassador-agent service.
	// any tests that do any more complicated assertions should live in ambassador.git/pkg/agent
	for i := 0; i < 15; i++ {
		podName, err := getFakeAgentComPodName(ctx, cli)
		assert.NoError(t, err)

		localSnapshot := fmt.Sprintf("%s/snapshot.json", t.TempDir())
		time.Sleep(time.Second * time.Duration(i))
		if err := dexec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "cp", podName+":/tmp/snapshot.json", localSnapshot).Run(); err != nil {
			t.Logf("Error running kubectl cp: %+v", err)
			continue
		}
		if _, err := os.Stat(localSnapshot); os.IsNotExist(err) {
			t.Log("Could not copy file from agentcom, retrying...")
			continue
		}
		snapbytes, err := ioutil.ReadFile(localSnapshot)
		if err != nil {
			t.Logf("Error reading snapshot file: %+v", err)
			continue
		}
		found = true

		err = json.Unmarshal(snapbytes, reportSnapshot)
		if err != nil {
			t.Fatal("Could not unmarshal report snapshot")
		}

		err = json.Unmarshal(reportSnapshot.RawSnapshot, ambSnapshot)
		if err != nil {
			t.Fatal("Could not unmarshal ambassador snapshot")
		}
		if !snapshotIsSane(ambSnapshot, t, waitArgo) {
			continue
		}
		break
	}
	require.True(t, found, "Could not cp file from agentcom")
	return reportSnapshot, ambSnapshot
}

func snapshotIsSane(ambSnapshot *snapshotTypes.Snapshot, t *testing.T, hasArgo bool) bool {
	if ambSnapshot.Kubernetes == nil {
		t.Log("K8s snapshot empty, retrying")
		return false
	}
	if len(ambSnapshot.Kubernetes.Services) == 0 {
		t.Log("K8s snapshot services empty, retrying")
		return false
	}
	if len(ambSnapshot.Kubernetes.Mappings) == 0 {
		t.Log("K8s snapshot mappings empty, retrying")
		return false
	}
	if len(ambSnapshot.Kubernetes.Pods) == 0 {
		t.Log("K8s snapshot pods empty, retrying")
		return false
	}
	if hasArgo && len(ambSnapshot.Kubernetes.ArgoRollouts) == 0 {
		t.Log("K8s snapshot argo rollouts empty, retrying")
		return false
	}
	if hasArgo && len(ambSnapshot.Kubernetes.ArgoApplications) == 0 {
		t.Log("K8s snapshot argo applications empty, retrying")
		return false
	}
	if !hasArgo && len(ambSnapshot.Kubernetes.ArgoRollouts) != 0 {
		t.Log("K8s snapshot argo rollouts should be empty, retrying")
		return false
	}
	if !hasArgo && len(ambSnapshot.Kubernetes.ArgoApplications) != 0 {
		t.Log("K8s snapshot argo applications should be empty, retrying")
		return false
	}

	return true
}
func applyArgoResources(t *testing.T, ctx context.Context, kubeconfig string, cli *kates.Client) {
	kubeinfo := k8s.NewKubeInfo(kubeconfig, "", "")
	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, time.Minute, true, false, "./testdata/argo-rollouts-crd.yaml"))
	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, time.Minute, true, false, "./testdata/argo-application-crd.yaml"))
	time.Sleep(3 * time.Second)
	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, time.Minute, true, false, "./testdata/argo-rollouts.yaml"))
	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, time.Minute, true, false, "./testdata/argo-application.yaml"))
}

func needsDockerBuilds(ctx context.Context, var2file map[string]string) error {
	var targets []string
	for varname, filename := range var2file {
		if os.Getenv(varname) == "" {
			targets = append(targets, filename)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	if os.Getenv("DEV_REGISTRY") == "" {
		registry := dtest.DockerRegistry(ctx)
		os.Setenv("DEV_REGISTRY", registry)
		os.Setenv("DTEST_REGISTRY", registry)
	}
	cmdline := append([]string{"make", "-C", "../.."}, targets...)
	if err := dexec.CommandContext(ctx, cmdline[0], cmdline[1:]...).Run(); err != nil {
		return err
	}
	for varname, filename := range var2file {
		if os.Getenv(varname) == "" {
			dat, err := ioutil.ReadFile(filepath.Join("../..", filename))
			if err != nil {
				return err
			}
			lines := strings.Split(strings.TrimSpace(string(dat)), "\n")
			if len(lines) < 2 {
				return fmt.Errorf("malformed docker.mk tagfile %q", filename)
			}
			if err := os.Setenv(varname, lines[1]); err != nil {
				return err
			}
		}
	}
	return nil
}

func yamlFilename(t *testing.T, inFilename, image string) string {
	dat, err := ioutil.ReadFile(inFilename)
	require.NoError(t, err)
	dat = bytes.ReplaceAll(dat, []byte("$imageRepo$:$version$"), []byte(image))
	outFilename := filepath.Join(t.TempDir(), strings.TrimSuffix(filepath.Base(inFilename), ".in"))
	require.NoError(t, ioutil.WriteFile(outFilename, dat, 0644))
	return outFilename
}

func setup(t *testing.T, ctx context.Context, kubeconfig string, cli *kates.Client) {
	require.NoError(t, needsDockerBuilds(ctx, map[string]string{
		"AMBASSADOR_DOCKER_IMAGE": "docker/emissary.docker.push.remote",
		"KAT_SERVER_DOCKER_IMAGE": "docker/kat-server.docker.push.remote",
	}))

	image := os.Getenv("AMBASSADOR_DOCKER_IMAGE")
	require.NotEmpty(t, image)

	crdFile := yamlFilename(t, "../../manifests/emissary/emissary-crds.yaml.in", image)
	aesFile := yamlFilename(t, "../../manifests/emissary/emissary-emissaryns.yaml.in", image)

	kubeinfo := k8s.NewKubeInfo(kubeconfig, "", "")

	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, time.Minute, true, false, crdFile))
	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, time.Minute, true, false, "./testdata/namespace.yaml"))
	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, 2*time.Minute, true, false, aesFile))
	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, 2*time.Minute, true, false, "./testdata/fake-agentcom.yaml"))

	dep := &kates.Deployment{
		TypeMeta: kates.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      "emissary-ingress-agent",
			Namespace: "emissary",
		},
	}

	patch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"env": []interface{}{
								map[string]interface{}{
									"name":  "RPC_CONNECTION_ADDRESS",
									"value": "http://agentcom-server.default:8080/",
								},
							},
							"name": "agent",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NoError(t, cli.Patch(ctx, dep, kates.StrategicMergePatchType, []byte(patch), dep))

	time.Sleep(3 * time.Second)
	require.NoError(t, kubeapply.Kubeapply(ctx, kubeinfo, time.Minute, true, false, "./testdata/sample-config.yaml"))
}

func deleteArgoResources(t *testing.T, ctx context.Context, kubeconfig string) {
	// cleaning up argo crds so the e2e test can be deterministic
	cmd := dexec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "delete", "crd", "--ignore-not-found=true", "rollouts.argoproj.io")
	out, err := cmd.CombinedOutput()
	t.Log(fmt.Sprintf("Kubectl delete crd rollouts output: %s", out))
	if err != nil {
		t.Errorf("Error running kubectl delete crd rollouts: %s", err)
	}
	cmd = dexec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "delete", "crd", "--ignore-not-found=true", "applications.argoproj.io")
	out, err = cmd.CombinedOutput()
	t.Log(fmt.Sprintf("Kubectl delete crd applications output: %s", out))
	if err != nil {
		t.Errorf("Error running kubectl delete crd applications: %s", err)
	}
}

func getFakeAgentComPodName(ctx context.Context, cli *kates.Client) (string, error) {
	query := kates.Query{
		Kind:          "Pod",
		LabelSelector: "app=agentcom-server",
		Namespace:     "default",
	}
	pods := []*kates.Pod{}
	err := cli.List(ctx, query, &pods)
	if err != nil {
		return "", err
	}
	if len(pods) < 1 {
		return "", errors.New("No pods found with label app=agentcom-server")
	}
	return pods[0].ObjectMeta.Name, nil
}
