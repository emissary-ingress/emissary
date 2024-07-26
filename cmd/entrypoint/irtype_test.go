package entrypoint_test

import (
	"encoding/json"
	"testing"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
	"github.com/stretchr/testify/assert"
)

func TestIRRouteWeight(t *testing.T) {
	rw := entrypoint.IRRouteWeight{
		{Int: 1},
		{Str: "foo"},
		{Int: 2},
	}

	// MarshalJSON
	expectedJSON := `[1,"foo",2]`
	actualJSON, err := json.Marshal(rw)
	assert.Nil(t, err)
	assert.Equal(t, expectedJSON, string(actualJSON))

	var check entrypoint.IRRouteWeight
	err = json.Unmarshal(actualJSON, &check)
	assert.Nil(t, err)
	assert.Equal(t, rw, check)

	// UnmarshalJSON
	jsonData := []byte(`[1,"bar",3]`)
	expectedRW := entrypoint.IRRouteWeight{
		{Int: 1},
		{Str: "bar"},
		{Int: 3},
	}
	var actualRW entrypoint.IRRouteWeight
	err = json.Unmarshal(jsonData, &actualRW)
	assert.Nil(t, err)
	assert.Equal(t, expectedRW, actualRW)
}

func TestIRCluster(t *testing.T) {
	clusterJSON := `{
		"_active": true,
		"_cache_key": "Cluster-cluster_127_0_0_1_8877_default",
		"_errored": false,
		"_hostname": "127.0.0.1",
		"_namespace": "default",
		"_port": 8877,
		"_referenced_by": [
		    "--internal--"
		],
		"_resolver": "kubernetes-service",
		"_rkey": "cluster_127_0_0_1_8877_default",
		"connect_timeout_ms": 3000,
		"enable_endpoints": false,
		"enable_ipv4": true,
		"enable_ipv6": false,
		"envoy_name": "cluster_127_0_0_1_8877_default",
		"health_checks": {
			"_active": true,
			"_errored": false,
			"_rkey": "ir.health_checks",
			"kind": "IRHealthChecks",
			"location": "--internal--",
			"name": "health_checks",
			"namespace": "default"
		},
		"ignore_cluster": false,
		"kind": "IRCluster",
		"lb_type": "round_robin",
		"location": "--internal--",
		"name": "cluster_127_0_0_1_8877_default",
		"namespace": "default",
		"respect_dns_ttl": false,
		"service": "127.0.0.1:8877",
		"stats_name": "127_0_0_1_8877",
		"targets": [
			{
				"ip": "127.0.0.1",
				"port": 8877,
				"target_kind": "IPaddr"
			}
		],
		"type": "strict_dns",
		"urls": [
			"tcp://127.0.0.1:8877"
		]
	}`

	expectedCluster := entrypoint.IRCluster{
		IRResource: entrypoint.IRResource{
			Active:       true,
			CacheKey:     "Cluster-cluster_127_0_0_1_8877_default",
			Errored:      false,
			ReferencedBy: []string{"--internal--"},
			RKey:         "cluster_127_0_0_1_8877_default",
			Location:     "--internal--",
			Kind:         "IRCluster",
			Name:         "cluster_127_0_0_1_8877_default",
			Namespace:    "default",
		},
		BarHostname:      "127.0.0.1",
		BarNamespace:     "default",
		Port:             8877,
		Resolver:         "kubernetes-service",
		ConnectTimeoutMs: 3000,
		EnableEndpoints:  false,
		EnableIPv4:       true,
		EnableIPv6:       false,
		EnvoyName:        "cluster_127_0_0_1_8877_default",
		HealthChecks: entrypoint.IRClusterHealthCheck{
			IRResource: entrypoint.IRResource{
				Active:    true,
				Errored:   false,
				RKey:      "ir.health_checks",
				Kind:      "IRHealthChecks",
				Location:  "--internal--",
				Name:      "health_checks",
				Namespace: "default",
			},
		},
		IgnoreCluster: false,
		LBType:        "round_robin",
		RespectDNSTTL: false,
		Service:       "127.0.0.1:8877",
		StatsName:     "127_0_0_1_8877",
		Targets: []entrypoint.IRClusterTarget{
			{
				IP:         "127.0.0.1",
				Port:       8877,
				TargetKind: "IPaddr",
			},
		},
		Type: "strict_dns",
		URLs: []string{
			"tcp://127.0.0.1:8877",
		},
	}

	var unmarshaledCluster entrypoint.IRCluster
	err := json.Unmarshal([]byte(clusterJSON), &unmarshaledCluster)
	assert.Nil(t, err)
	assert.Equal(t, expectedCluster, unmarshaledCluster)

	remarshaledJSON, err := json.Marshal(unmarshaledCluster)
	assert.Nil(t, err)

	var unmarshaledCluster2 entrypoint.IRCluster
	err = json.Unmarshal(remarshaledJSON, &unmarshaledCluster2)
	assert.Nil(t, err)
	assert.Equal(t, expectedCluster, unmarshaledCluster2)
}
