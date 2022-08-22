package entrypoint_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
	bootstrap "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/bootstrap/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

func getExpected(expectedFile string, inputObjects []kates.Object) ([]RenderedListener, []v3alpha1.Mapping, []string, error) {
	// Figure out all the mappings and clusters we'll need.
	neededClusters := []string{}
	neededMappings := []v3alpha1.Mapping{}

	// Read the expected rendering from a file.
	content, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		return nil, nil, nil, err
	}

	var expectedListeners []RenderedListener
	if err := json.Unmarshal(content, &expectedListeners); err != nil {
		return nil, nil, nil, err
	}

	// Build the set of expected mappings and clusters from our objects.
	clusterRE := regexp.MustCompile("[^0-9A-Za-z_]")

	for _, obj := range inputObjects {
		// Skip things that aren't Mappings.
		mapping, ok := obj.(*v3alpha1.Mapping)

		if !ok {
			continue
		}

		// We need to see this Mapping in our snapshot.
		neededMappings = append(neededMappings, *mapping)

		// Grab the cluster name, and remember it for later.
		mangledService := clusterRE.ReplaceAll([]byte(mapping.Spec.Service), []byte("_"))
		clusterName := fmt.Sprintf("cluster_%s_default", mangledService)
		neededClusters = append(neededClusters, clusterName)
	}

	return expectedListeners, neededMappings, neededClusters, nil
}

func testSemanticSet(t *testing.T, inputFile string, expectedFile string) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)

	inputObjects, err := LoadYAML(inputFile)
	require.NoError(t, err)

	// expectedListeners is what we think we're going to get.
	expectedListeners, neededMappings, neededClusters, err := getExpected(expectedFile, inputObjects)
	require.NoError(t, err)
	expectedJSON, err := JSONifyRenderedListeners(expectedListeners)
	require.NoError(t, err)

	// Now, what did we _actually_ get?
	require.NoError(t, f.UpsertFile(inputFile))
	f.Flush()

	snap, err := f.GetSnapshot(func(snapshot *snapshot.Snapshot) bool {
		// XXX Ew. Switch to a dict, FFS.
		for _, mapping := range neededMappings {
			mappingNamespace := mapping.Namespace

			if mappingNamespace == "" {
				mappingNamespace = "default"
			}

			mappingName := mapping.Name

			t.Logf("GetSnapshot: looking for %s/%s", mappingNamespace, mappingName)

			found := false
			for _, m := range snapshot.Kubernetes.Mappings {
				if m.Namespace == mappingNamespace && m.Name == mappingName {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}

		return true
	})
	require.NoError(t, err)
	require.NotNil(t, snap)

	envoyConfig, err := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		for _, cluster := range neededClusters {
			if FindCluster(config, ClusterNameContains(cluster)) == nil {
				return false
			}
		}

		return true
	})
	require.NoError(t, err)
	require.NotNil(t, envoyConfig)

	actualListeners, err := RenderEnvoyConfig(t, envoyConfig)
	require.NoError(t, err)
	actualJSON, err := JSONifyRenderedListeners(actualListeners)
	require.NoError(t, err)

	err = ioutil.WriteFile("/tmp/host-semantics-expected.json", []byte(expectedJSON), 0644)
	if err == io.EOF {
		err = nil
	}
	require.NoError(t, err)

	err = ioutil.WriteFile("/tmp/host-semantics-actual.json", []byte(actualJSON), 0644)
	if err == io.EOF {
		err = nil
	}
	require.NoError(t, err)

	require.Equal(t, expectedJSON, actualJSON, "Mismatch!")
}

func TestHostSemanticsMinimal(t *testing.T) {
	testSemanticSet(t, "testdata/hostsem-minimal.yaml", "testdata/hostsem-minimal-expected.json")
}

func TestHostSemanticsBasic(t *testing.T) {
	testSemanticSet(t, "testdata/hostsem-basic.yaml", "testdata/hostsem-basic-expected.json")
}

func TestHostSemanticsCleartextOnly(t *testing.T) {
	testSemanticSet(t, "testdata/hostsem-cleartextonly.yaml", "testdata/hostsem-cleartextonly-expected.json")
}

func TestHostSemanticsDisjoint(t *testing.T) {
	testSemanticSet(t, "testdata/hostsem-disjoint-hosts.yaml", "testdata/hostsem-disjoint-hosts-expected.json")
}

func TestHostSemanticsHostSelector(t *testing.T) {
	testSemanticSet(t, "testdata/hostsem-hostsel.yaml", "testdata/hostsem-hostsel-expected.json")
}
