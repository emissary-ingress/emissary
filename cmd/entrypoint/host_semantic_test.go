package entrypoint_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	"github.com/datawire/ambassador/v2/internal/pkg/testutils"
	bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/stretchr/testify/require"
)

func getExpected(expectedFile string, inputObjects []kates.Object) ([]testutils.RenderedListener, []v3alpha1.Mapping, []string) {
	// Figure out all the mappings and clusters we'll need.
	neededClusters := []string{}
	neededMappings := []v3alpha1.Mapping{}

	// Read the expected rendering from a file.
	content, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		panic(err)
	}

	var expectedListeners []testutils.RenderedListener
	err = json.Unmarshal(content, &expectedListeners)

	if err != nil {
		panic(err)
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

		// fmt.Printf("CHECK Mapping %s\n%s\n", mapping.Name, testutils.JSONify(mapping))

		// Grab the cluster name, and remember it for later.
		mangledService := clusterRE.ReplaceAll([]byte(mapping.Spec.Service), []byte("_"))
		clusterName := fmt.Sprintf("cluster_%s_default", mangledService)
		neededClusters = append(neededClusters, clusterName)
	}

	return expectedListeners, neededMappings, neededClusters
}

func testSemanticSet(t *testing.T, inputFile string, expectedFile string) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: false}, nil)

	inputObjects := testutils.LoadYAML(inputFile)

	// expectedListeners is what we think we're going to get.
	expectedListeners, neededMappings, neededClusters := getExpected(expectedFile, inputObjects)
	expectedJSON := testutils.JSONifyRenderedListeners(expectedListeners)

	// Now, what did we _actually_ get?
	f.UpsertFile(inputFile)
	f.Flush()

	snap := f.GetSnapshot(func(snapshot *snapshot.Snapshot) bool {
		// XXX Ew. Switch to a dict, FFS.
		for _, mapping := range neededMappings {
			mappingNamespace := mapping.Namespace

			if mappingNamespace == "" {
				mappingNamespace = "default"
			}

			mappingName := mapping.Name

			fmt.Printf("GetSnapshot: looking for %s/%s\n", mappingNamespace, mappingName)

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

	require.NotNil(t, snap)

	envoyConfig := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		for _, cluster := range neededClusters {
			if FindCluster(config, ClusterNameContains(cluster)) == nil {
				return false
			}
		}

		return true
	})

	require.NotNil(t, envoyConfig)

	actualListeners := testutils.RenderEnvoyConfig(envoyConfig)
	actualJSON := testutils.JSONifyRenderedListeners(actualListeners)

	err := ioutil.WriteFile("/tmp/host-semantics-expected.json", []byte(expectedJSON), 0644)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("/tmp/host-semantics-actual.json", []byte(actualJSON), 0644)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		panic(err)
	}

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
