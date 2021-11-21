package v2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	v2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	v3alpha1 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
)

func TestConvertAuthService(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "authsvc.json"))
	require.NoError(t, err)

	var v2AuthServices []v2.AuthService

	err = json.Unmarshal(bytes, &v2AuthServices)
	require.NoError(t, err)

	for _, v2authsvc := range v2AuthServices {
		var v3authsvc v3alpha1.AuthService

		err := v3authsvc.ConvertFrom(&v2authsvc)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2authsvc, "", "  ")
		v3bytes, err := json.MarshalIndent(v3authsvc, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}

func TestConvertDevPortal(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "devportals.json"))
	require.NoError(t, err)

	var v2DevPortals []v2.DevPortal

	err = json.Unmarshal(bytes, &v2DevPortals)
	require.NoError(t, err)

	for _, v2devportal := range v2DevPortals {
		var v3devportal v3alpha1.DevPortal

		err := v3devportal.ConvertFrom(&v2devportal)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2devportal, "", "  ")
		v3bytes, err := json.MarshalIndent(v3devportal, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}

func TestConvertHost(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "hosts.json"))
	require.NoError(t, err)

	var v2Hosts []v2.Host

	err = json.Unmarshal(bytes, &v2Hosts)
	require.NoError(t, err)

	for _, v2host := range v2Hosts {
		var v3host v3alpha1.Host

		err := v3host.ConvertFrom(&v2host)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2host, "", "  ")
		v3bytes, err := json.MarshalIndent(v3host, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}

func TestConvertLogService(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "logsvc.json"))
	require.NoError(t, err)

	var v2LogServices []v2.LogService

	err = json.Unmarshal(bytes, &v2LogServices)
	require.NoError(t, err)

	for _, v2logservice := range v2LogServices {
		var v3logservice v3alpha1.LogService

		err := v3logservice.ConvertFrom(&v2logservice)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2logservice, "", "  ")
		v3bytes, err := json.MarshalIndent(v3logservice, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}

func TestConvertMapping(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "mappings.json"))
	require.NoError(t, err)

	var v2Mappings []v2.Mapping

	err = json.Unmarshal(bytes, &v2Mappings)
	require.NoError(t, err)

	for _, v2mapping := range v2Mappings {
		var v3mapping v3alpha1.Mapping

		err := v3mapping.ConvertFrom(&v2mapping)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2mapping, "", "  ")
		v3bytes, err := json.MarshalIndent(v3mapping, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}

func TestConvertRateLimitService(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "ratelimitsvc.json"))
	require.NoError(t, err)

	var v2RateLimitServices []v2.RateLimitService

	err = json.Unmarshal(bytes, &v2RateLimitServices)
	require.NoError(t, err)

	for _, v2ratelimitservice := range v2RateLimitServices {
		var v3ratelimitservice v3alpha1.RateLimitService

		err := v3ratelimitservice.ConvertFrom(&v2ratelimitservice)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2ratelimitservice, "", "  ")
		v3bytes, err := json.MarshalIndent(v3ratelimitservice, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
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
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "tcpmappings.json"))
	require.NoError(t, err)

	var v2TCPMappings []v2.TCPMapping

	err = json.Unmarshal(bytes, &v2TCPMappings)
	require.NoError(t, err)

	for _, v2tcpmapping := range v2TCPMappings {
		var v3tcpmapping v3alpha1.TCPMapping

		err := v3tcpmapping.ConvertFrom(&v2tcpmapping)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2tcpmapping, "", "  ")
		v3bytes, err := json.MarshalIndent(v3tcpmapping, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}

func TestConvertTLSContext(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "tlscontexts.json"))
	require.NoError(t, err)

	var v2TLSContexts []v2.TLSContext

	err = json.Unmarshal(bytes, &v2TLSContexts)
	require.NoError(t, err)

	for _, v2tlscontext := range v2TLSContexts {
		var v3tlscontext v3alpha1.TLSContext

		err := v3tlscontext.ConvertFrom(&v2tlscontext)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2tlscontext, "", "  ")
		v3bytes, err := json.MarshalIndent(v3tlscontext, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}

func TestConvertTracingService(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "tracingsvc.json"))
	require.NoError(t, err)

	var v2TracingServices []v2.TracingService

	err = json.Unmarshal(bytes, &v2TracingServices)
	require.NoError(t, err)

	for _, v2tracingservice := range v2TracingServices {
		var v3tracingservice v3alpha1.TracingService

		err := v3tracingservice.ConvertFrom(&v2tracingservice)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2tracingservice, "", "  ")
		v3bytes, err := json.MarshalIndent(v3tracingservice, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}
