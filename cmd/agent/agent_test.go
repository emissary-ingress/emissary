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

	"github.com/datawire/dlib/dexec"
	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/pkg/api/agent"
	"github.com/datawire/ambassador/pkg/dtest"
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/kubeapply"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
)

// This test is supposed to be a very lightweight end to end test.
// We're essentially testing that the k8s yaml configuration allows the agent to report on all the
// things the cloud app needs. We do this with dtest, which spins up a k3ds cluster by default, or
// you can point it at your own cluster by running `go test` with env var `DTEST_KUBECONFIG=$HOME/.kube/config`
// More complicated business logic tests live in ambassador.git/pkg/agent
func TestAgentE2E(t *testing.T) {
	kubeconfig := dtest.Kubeconfig()
	cli, err := kates.NewClient(kates.ClientConfig{Kubeconfig: kubeconfig})
	assert.Nil(t, err)
	// applies all k8s yaml to dtest cluter
	// ambassador, ambassador-agent, rbac, crds, and a fake agentcom that implements the grpc
	// server for the agent
	setup(t, kubeconfig, cli)

	// eh lets make sure the agent came up
	time.Sleep(time.Second * 3)

	podName, err := getFakeAgentComPodName(cli)
	assert.Nil(t, err)

	podFile := fmt.Sprintf("%s:%s", podName, "/tmp/snapshot.json")
	localSnapshot := fmt.Sprintf("%s/snapshot.json", t.TempDir())

	found := false
	reportSnapshot := &agent.Snapshot{}
	ambSnapshot := &snapshotTypes.Snapshot{}
	ctx := context.Background()
	// now we're going to go copy the snapshot.json file from our fake agentcom
	// when the agentcom gets a snapshot from the agent, it'll store it at /tmp/snapshot.json
	// we do this in a loop because it might take ambassador and the agent a sec to get into the
	// state we're asserting. this is okay, this test is just to make sure that the agent RBAC
	// is correct and that the agent can talk to the ambassador-agent service.
	// any tests that do any more complicated assertions should live in ambassador.git/pkg/agent
	for i := 0; i < 10; i++ {
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
		if !snapshotIsSane(ambSnapshot, t) {
			continue
		}
		break
	}
	assert.True(t, found, "Could not cp file from agentcom")

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
}

func snapshotIsSane(ambSnapshot *snapshotTypes.Snapshot, t *testing.T) bool {
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

	return true
}

func setup(t *testing.T, kubeconfig string, cli *kates.Client) {
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
	ctx := context.Background()

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

func getFakeAgentComPodName(cli *kates.Client) (string, error) {
	query := kates.Query{
		Kind:          "Pod",
		LabelSelector: "app=agentcom-server",
		Namespace:     "default",
	}
	pods := []*kates.Pod{}
	err := cli.List(context.Background(), query, &pods)
	if err != nil {
		return "", err
	}
	if len(pods) < 1 {
		return "", errors.New("No pods found with label app=agentcom-server")
	}
	return pods[0].ObjectMeta.Name, nil
}
