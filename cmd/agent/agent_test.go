package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/v2/pkg/api/agent"
	"github.com/datawire/ambassador/v2/pkg/dtest"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/kubeapply"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
)

// This test is supposed to be a very lightweight end to end test.
// We're essentially testing that the k8s yaml configuration allows the agent to report on all the
// things the cloud app needs. We do this with dtest, which spins up a k3ds cluster by default, or
// you can point it at your own cluster by running `go test` with env var `DTEST_KUBECONFIG=$HOME/.kube/config`
// More complicated business logic tests live in ambassador.git/pkg/agent
func TestAgentE2E(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)
	kubeconfig := dtest.Kubeconfig(ctx)
	cli, err := kates.NewClient(kates.ClientConfig{Kubeconfig: kubeconfig})
	assert.Nil(t, err)
	// applies all k8s yaml to dtest cluter
	// ambassador, ambassador-agent, rbac, crds, and a fake agentcom that implements the grpc
	// server for the agent
	setup(t, ctx, kubeconfig, cli)
	defer deleteArgoResources(t, ctx, kubeconfig)

	// eh lets make sure the agent came up
	time.Sleep(time.Second * 3)

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

	applyArgoResources(t, kubeconfig, cli)
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
		assert.Nil(t, err)

		podFile := fmt.Sprintf("%s:%s", podName, "/tmp/snapshot.json")
		localSnapshot := fmt.Sprintf("%s/snapshot.json", t.TempDir())
		time.Sleep(time.Second * time.Duration(i))
		cmd := dexec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "cp", podFile, localSnapshot)
		out, err := cmd.CombinedOutput()
		t.Log(fmt.Sprintf("Kubectl cp output: %s", out))
		if err != nil {
			t.Log(fmt.Sprintf("Error running kubectl cp: %+v", err))
			continue
		}
		if _, err := os.Stat(localSnapshot); os.IsNotExist(err) {
			t.Log("Could not copy file from agentcom, retrying...")
			continue
		}
		snapbytes, err := ioutil.ReadFile(localSnapshot)
		if err != nil {
			t.Log(fmt.Sprintf("Error reading snapshot file: %+v", err))
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
	assert.True(t, found, "Could not cp file from agentcom")
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
func applyArgoResources(t *testing.T, kubeconfig string, cli *kates.Client) {
	kubeinfo := k8s.NewKubeInfo(kubeconfig, "", "")
	err := kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, "./test/argo-rollouts-crd.yaml")
	assert.Nil(t, err)
	err = kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, "./test/argo-rollouts.yaml")
	assert.Nil(t, err)
	err = kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, "./test/argo-application-crd.yaml")
	assert.Nil(t, err)
	err = kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, "./test/argo-application.yaml")
	assert.Nil(t, err)
}

func setup(t *testing.T, ctx context.Context, kubeconfig string, cli *kates.Client) {
	// okay, yes this is gross, but we're revamping all the yaml right now, so i'm just making
	// this as frictionless as possible for the time being
	// TODO(acookin): this will probably need to change when we finish #1280
	yamlPath := "../../docs/yaml/"
	crdFile := yamlPath + "ambassador/ambassador-crds.yaml"
	aesFile := yamlPath + "aes.yaml"
	aesDat, err := ioutil.ReadFile(aesFile)
	assert.Nil(t, err)
	image := os.Getenv("AMBASSADOR_DOCKER_IMAGE")
	assert.NotEmpty(t, image)

	aesReplaced := strings.ReplaceAll(string(aesDat), "docker.io/datawire/aes:$version$", image)
	newAesFile := t.TempDir() + "/aes.yaml"

	err = ioutil.WriteFile(newAesFile, []byte(aesReplaced), 0644)
	assert.Nil(t, err)
	kubeinfo := k8s.NewKubeInfo(kubeconfig, "", "")

	err = kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, crdFile)
	assert.Nil(t, err)
	err = kubeapply.Kubeapply(kubeinfo, time.Second*120, true, false, newAesFile)
	assert.Nil(t, err)
	err = kubeapply.Kubeapply(kubeinfo, time.Second*120, true, false, "./fake-agentcom.yaml")
	assert.Nil(t, err)

	dep := &kates.Deployment{
		TypeMeta: kates.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      "ambassador-agent",
			Namespace: "ambassador",
		},
	}

	patch := fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"agent","env":[{"name":"%s", "value":"%s"}]}]}}}}`, "RPC_CONNECTION_ADDRESS", "http://agentcom-server.default:8080/")
	err = cli.Patch(ctx, dep, kates.StrategicMergePatchType, []byte(patch), dep)
	assert.Nil(t, err)
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
