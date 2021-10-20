package k8s_resource_types_test

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/kates/k8s_resource_types"
)

func testVersionEquiv(t *testing.T, filename, expectedAPIVersion string, convert func(kates.Object) (kates.Object, error)) {
	testdata := func() map[string]string {
		file, err := os.Open(filepath.Join("testdata", filename))
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, file.Close())
		}()

		ret := make(map[string]string)

		yr := utilyaml.NewYAMLReader(bufio.NewReader(file))
		for i := 0; true; i++ {
			bs, err := yr.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
			}
			str := strings.TrimSpace(strings.TrimPrefix(string(bs), "---\n"))

			require.True(t, strings.HasPrefix(str, "apiVersion: "), "YAML document[%d] starts with apiVersion: %q", i, str)
			apiVersion := strings.TrimPrefix(strings.SplitN(str, "\n", 2)[0], "apiVersion: ")
			ret[apiVersion] = str
		}
		return ret
	}()

	expectedOut, ok := testdata[expectedAPIVersion]
	require.True(t, ok)

	for testname, testyaml := range testdata {
		testname, testyaml := testname, testyaml
		t.Run(testname, func(t *testing.T) {
			var untyped kates.Unstructured
			require.NoError(t, yaml.UnmarshalStrict([]byte(testyaml), &untyped))
			typed, err := convert(&untyped)
			require.NoError(t, err)

			actualOutBytes, err := yaml.Marshal(typed)
			assert.NoError(t, err)

			actualOut := strings.TrimSpace(string(actualOutBytes))
			assert.Equal(t, expectedOut, actualOut)
		})
	}
}

func TestNewIngress(t *testing.T) {
	testVersionEquiv(t, "ingress.yaml", "extensions/v1beta1",
		func(in kates.Object) (kates.Object, error) {
			return k8s_resource_types.NewIngress(in)
		})
}

func TestNewIngressClass(t *testing.T) {
	testVersionEquiv(t, "ingressclass.yaml", "networking.k8s.io/v1",
		func(in kates.Object) (kates.Object, error) {
			return k8s_resource_types.NewIngressClass(in)
		})
}
