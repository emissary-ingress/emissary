package kubeapply_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/datawire/ambassador/pkg/dtest"
	"github.com/datawire/ambassador/pkg/kubeapply"
	"github.com/datawire/dlib/dlog"
)

func TestDocker(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip(err)
	}

	if os.Getenv("DOCKER_REGISTRY") == "" {
		os.Setenv("DOCKER_REGISTRY", dtest.DockerRegistry(ctx))
	}

	_, err := kubeapply.ExpandResource("docker.yaml")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
