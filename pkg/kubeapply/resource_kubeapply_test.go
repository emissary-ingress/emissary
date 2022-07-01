package kubeapply_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/dtest"
	"github.com/emissary-ingress/emissary/v3/pkg/kubeapply"
)

func needsDocker(t *testing.T) {
	if _, err := dexec.LookPath("docker"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("This should not happen in CI: skipping test because 'docker' is not installed: %v", err)
		}
		t.Skip(err)
	}
}

func TestDocker(t *testing.T) {
	needsDocker(t)

	ctx := dlog.NewTestContext(t, false)

	if os.Getenv("DOCKER_REGISTRY") == "" {
		os.Setenv("DOCKER_REGISTRY", dtest.DockerRegistry(ctx))
	}

	_, err := kubeapply.ExpandResource(ctx, "testdata/docker.yaml")
	assert.NoError(t, err)
}

func TestExpand(t *testing.T) {
	needsDocker(t)

	ctx := dlog.NewTestContext(t, false)

	if os.Getenv("DOCKER_REGISTRY") == "" {
		os.Setenv("DOCKER_REGISTRY", dtest.DockerRegistry(ctx))
	}

	outfiles, err := kubeapply.TestExpand(ctx, []string{"testdata/docker.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"testdata/docker.yaml.o"}, outfiles)

	actBytes, err := os.ReadFile("testdata/docker.yaml.o")
	assert.NoError(t, err)
	actStr := string(actBytes)

	img, err := kubeapply.TestImage(ctx, ".", "../../docker/test-http/Dockerfile")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(img, os.Getenv("DOCKER_REGISTRY")+"/kubeapply:"))

	expBytes, err := os.ReadFile("testdata/docker.yaml.o.exp")
	assert.NoError(t, err)
	expStr := strings.ReplaceAll(string(expBytes), "@IMAGE@", img)

	assert.Equal(t, expStr, actStr)
}
