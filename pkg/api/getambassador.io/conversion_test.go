package v2

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	v2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	v3alpha1 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
)

func TestConvertAuthService(t *testing.T) {
	var v2 []v2.AuthService
	var v3 []v3alpha1.AuthService

	testConvert(t, "authsvc", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.AuthService, 0, len(v2))

		for _, v2obj := range v2 {
			var v3obj v3alpha1.AuthService
			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

func TestConvertDevPortal(t *testing.T) {
	var v2 []v2.DevPortal
	var v3 []v3alpha1.DevPortal

	testConvert(t, "devportals", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.DevPortal, 0, len(v2))

		for _, v2obj := range v2 {
			var v3obj v3alpha1.DevPortal
			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

func TestConvertHost(t *testing.T) {
	var v2 []v2.Host
	var v3 []v3alpha1.Host

	testConvert(t, "host", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.Host, 0, len(v2))

		for _, v2obj := range v2 {
			var v3obj v3alpha1.Host
			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

func TestConvertLogService(t *testing.T) {
	var v2 []v2.LogService
	var v3 []v3alpha1.LogService

	testConvert(t, "logsvc", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.LogService, 0, len(v2))

		for _, v2obj := range v2 {
			var v3obj v3alpha1.LogService
			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

func TestConvertMapping(t *testing.T) {
	var v2 []v2.Mapping
	var v3 []v3alpha1.Mapping

	testConvert(t, "mappings", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.Mapping, 0, len(v2))

		for _, v2obj := range v2 {
			var v3obj v3alpha1.Mapping
			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

func TestConvertRateLimitService(t *testing.T) {
	var v2 []v2.RateLimitService
	var v3 []v3alpha1.RateLimitService

	testConvert(t, "ratelimitsvc", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.RateLimitService, 0, len(v2))

		for _, v2obj := range v2 {
			var v3obj v3alpha1.RateLimitService
			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

// func TestConvertKubernetesEndpointResolver(t *testing.T) {
// 	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "kubernetesendpointresolvers.json"))
// 	require.NoError(t, err)

// 	var v2KubernetesEndpointResolvers []v2.KubernetesEndpointResolver

// 	err = json.Unmarshal(bytes, &v2KubernetesEndpointResolvers)
// 	require.NoError(t, err)

// 	for _, v2kubernetesendpointresolver := range v2KubernetesEndpointResolvers {
// 		var v3kubernetesendpointresolver v3alpha1.KubernetesEndpointResolver

// 		err := v3kubernetesendpointresolver.ConvertFrom(&v2kubernetesendpointresolver)
// 		require.NoError(t, err)

// 		v2bytes, _ := json.MarshalIndent(v2kubernetesendpointresolver, "", "  ")
// 		v3bytes, err := json.MarshalIndent(v3kubernetesendpointresolver, "", "  ")
// 		require.NoError(t, err)

// 		fmt.Printf("V2: %s\n", string(v2bytes))
// 		fmt.Printf("V3: %s\n", string(v3bytes))
// 	}
// }

// func TestConvertKubernetesServiceResolver(t *testing.T) {
// 	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "kubernetesserviceresolvers.json"))
// 	require.NoError(t, err)

// 	var v2KubernetesServiceResolvers []v2.KubernetesServiceResolver

// 	err = json.Unmarshal(bytes, &v2KubernetesServiceResolvers)
// 	require.NoError(t, err)

// 	for _, v2kubernetesserviceresolver := range v2KubernetesServiceResolvers {
// 		var v3kubernetesserviceresolver v3alpha1.KubernetesServiceResolver

// 		err := v3kubernetesserviceresolver.ConvertFrom(&v2kubernetesserviceresolver)
// 		require.NoError(t, err)

// 		v2bytes, _ := json.MarshalIndent(v2kubernetesserviceresolver, "", "  ")
// 		v3bytes, err := json.MarshalIndent(v3kubernetesserviceresolver, "", "  ")
// 		require.NoError(t, err)

// 		fmt.Printf("V2: %s\n", string(v2bytes))
// 		fmt.Printf("V3: %s\n", string(v3bytes))
// 	}
// }

// func TestConvertConsulResolver(t *testing.T) {
// 	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "consulresolvers.json"))
// 	require.NoError(t, err)

// 	var v2ConsulResolvers []v2.ConsulResolver

// 	err = json.Unmarshal(bytes, &v2ConsulResolvers)
// 	require.NoError(t, err)

// 	for _, v2consulresolver := range v2ConsulResolvers {
// 		var v3consulresolver v3alpha1.ConsulResolver

// 		err := v3consulresolver.ConvertFrom(&v2consulresolver)
// 		require.NoError(t, err)

// 		v2bytes, _ := json.MarshalIndent(v2consulresolver, "", "  ")
// 		v3bytes, err := json.MarshalIndent(v3consulresolver, "", "  ")
// 		require.NoError(t, err)

// 		fmt.Printf("V2: %s\n", string(v2bytes))
// 		fmt.Printf("V3: %s\n", string(v3bytes))
// 	}
// }

func TestConvertTCPMapping(t *testing.T) {
	var v2 []v2.TCPMapping
	var v3 []v3alpha1.TCPMapping

	testConvert(t, "tcpmappings", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.TCPMapping, 0, len(v2))
		for _, v2obj := range v2 {
			var v3obj v3alpha1.TCPMapping

			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

func TestConvertTLSContext(t *testing.T) {
	var v2 []v2.TLSContext
	var v3 []v3alpha1.TLSContext

	testConvert(t, "tlscontexts", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.TLSContext, 0, len(v2))

		for _, v2obj := range v2 {
			var v3obj v3alpha1.TLSContext
			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

func TestConvertTracingService(t *testing.T) {
	var v2 []v2.TracingService
	var v3 []v3alpha1.TracingService

	testConvert(t, "tracingsvc", &v2, &v3, func() interface{} {
		testObjects := make([]v3alpha1.TracingService, 0, len(v2))

		for _, v2obj := range v2 {
			var v3obj v3alpha1.TracingService
			require.NoError(t, v3obj.ConvertFrom(&v2obj))

			v3obj.APIVersion = "getambassador.io/v3alpha1"
			v3obj.Kind = v2obj.Kind
			testObjects = append(testObjects, v3obj)
		}

		return testObjects
	})
}

func reserialized(t *testing.T, obj interface{}) []byte {
	yamlbytes, err := yaml.Marshal(obj)
	require.NoError(t, err)

	var untyped interface{}
	err = yaml.Unmarshal(yamlbytes, &untyped)
	require.NoError(t, err)

	canonical, err := json.MarshalIndent(untyped, "", "\t")
	require.NoError(t, err)

	return canonical
}

func testConvert(t *testing.T, base string, v2list interface{}, v3list interface{}, getObjects func() interface{}) {
	v2jsonbytes, err := ioutil.ReadFile(path.Join("v2/testdata", base+".json"))
	require.NoError(t, err)

	v3yamlbytes, err := ioutil.ReadFile(path.Join("v3alpha1/testdata", base+".yaml"))
	require.NoError(t, err)

	err = json.Unmarshal(v2jsonbytes, v2list)
	require.NoError(t, err)

	// v3list is _not an untyped thing here_ -- we're just using interface{} to
	// dodge the type checking.
	err = yaml.Unmarshal(v3yamlbytes, v3list)
	require.NoError(t, err)

	v3jsonbytes := reserialized(t, v3list)
	v3checkbytes := reserialized(t, getObjects())

	if string(v3jsonbytes) != string(v3checkbytes) {
		t.Logf("v2bytes:\n%s\n", string(v2jsonbytes))
		t.Logf("v3bytes:\n%s\n", string(v3jsonbytes))
		t.Logf("v3check:\n%s\n", string(v3checkbytes))
	}

	require.Equal(t, string(v3jsonbytes), string(v3checkbytes))
}
