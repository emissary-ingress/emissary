package entrypoint

import (
	"encoding/json"
	"fmt"
)

type IRResource struct {
	Active       bool     `json:"_active"`
	CacheKey     string   `json:"_cache_key,omitempty"`
	Errored      bool     `json:"_errored"`
	ReferencedBy []string `json:"_referenced_by,omitempty"`
	RKey         string   `json:"_rkey,omitempty"`
	Location     string   `json:"location,omitempty"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace,omitempty"`
}

type IRClusterHealthCheck struct {
	IRResource
}

type IRClusterTarget struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	TargetKind string `json:"target_kind"`
}

type IRCluster struct {
	IRResource
	BarHostname      string               `json:"_hostname"`  // Why this _and_ hostname?
	BarNamespace     string               `json:"_namespace"` // Why this _and_ namespace?
	Port             int                  `json:"_port"`
	Resolver         string               `json:"_resolver"`
	ConnectTimeoutMs int                  `json:"connect_timeout_ms"`
	EnableEndpoints  bool                 `json:"enable_endpoints"`
	EnableIPv4       bool                 `json:"enable_ipv4"`
	EnableIPv6       bool                 `json:"enable_ipv6"`
	EnvoyName        string               `json:"envoy_name"`
	HealthChecks     IRClusterHealthCheck `json:"health_checks,omitempty"`
	IgnoreCluster    bool                 `json:"ignore_cluster"`
	LBType           string               `json:"lb_type"`
	RespectDNSTTL    bool                 `json:"respect_dns_ttl"`
	Service          string               `json:"service"`
	StatsName        string               `json:"stats_name"`
	Targets          []IRClusterTarget    `json:"targets"`
	Type             string               `json:"type"`
	URLs             []string             `json:"urls"`
}

type IRQueryParameter struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
	Regex bool   `json:"regex,omitempty"`
}

type IRRegexRewrite struct {
	Pattern      string `json:"pattern,omitempty"`
	Substitution string `json:"substitution,omitempty"`
}

// Route weights are really annoying: in Python they're a
// List[Union[str, int]], which is a pain to represent in Go.

type IRRouteWeightElement struct {
	Int int
	Str string
}

type IRRouteWeight []IRRouteWeightElement

func (rw IRRouteWeight) MarshalJSON() ([]byte, error) {
	arr := make([]interface{}, len(rw))

	for i, elem := range rw {
		if elem.Str != "" {
			arr[i] = elem.Str
		} else {
			arr[i] = elem.Int
		}
	}

	return json.Marshal(arr)
}

func (rw *IRRouteWeight) UnmarshalJSON(data []byte) error {
	var arr []interface{}

	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	*rw = make([]IRRouteWeightElement, len(arr))

	for i, elem := range arr {
		switch v := elem.(type) {
		case string:
			(*rw)[i] = IRRouteWeightElement{Str: v}
		case float64:
			(*rw)[i] = IRRouteWeightElement{Int: int(v)}
		default:
			return fmt.Errorf("unexpected type in IRRouteWeight: %T", elem)
		}
	}

	return nil
}

type IRMapping struct {
	IRResource
	Weight          int                `json:"_weight"`
	Cluster         IRCluster          `json:"cluster"`
	ClusterKey      string             `json:"cluster_key"`
	DefaultClass    string             `json:"default_class"`
	GroupID         string             `json:"group_id"`
	Headers         []IRHeader         `json:"headers"`
	Host            string             `json:"host"`
	Precedence      int                `json:"precedence"`
	Prefix          string             `json:"prefix"`
	QueryParameters []IRQueryParameter `json:"query_parameters,omitempty"`
	RegexRewrite    IRRegexRewrite     `json:"regex_rewrite,omitempty"`
	Resolver        string             `json:"resolver"`
	Rewrite         string             `json:"rewrite"`
	RouteWeight     IRRouteWeight      `json:"route_weight"`
	Service         string             `json:"service"`
	TimeoutMS       int                `json:"timeout_ms"`
}

type IRRequestPolicy struct {
	Action string `json:"action"`
}

type IRHost struct {
	IRResource
	Hostname       string                     `json:"hostname"`
	InsecureAction string                     `json:"insecure_action"`
	RequestPolicy  map[string]IRRequestPolicy `json:"requestPolicy"` // Yes, really.
	SecureAction   string                     `json:"secure_action"`
	SNI            string                     `json:"sni"`
}

type IRHeader struct {
	Name  string `json:"name"`
	Regex bool   `json:"regex"`
	Value string `json:"value"`
}

type IRGroup struct {
	IRResource
	DefaultClass    string             `json:"default_class"`
	GroupID         string             `json:"group_id"`
	GroupWeight     IRRouteWeight      `json:"group_weight"`
	Headers         []IRHeader         `json:"headers"`
	Host            string             `json:"host"`
	Mappings        []IRMapping        `json:"mappings"`
	Precedence      int                `json:"precedence"`
	Prefix          string             `json:"prefix"`
	QueryParameters []IRQueryParameter `json:"query_parameters"`
	RegexRewrite    IRRegexRewrite     `json:"regex_rewrite"`
	Rewrite         string             `json:"rewrite"`
	TimeoutMS       int                `json:"timeout_ms"`
}

type IR struct {
	Clusters map[string]IRCluster `json:"clusters"`
	Groups   []IRGroup            `json:"groups"`
	Hosts    []IRHost             `json:"hosts"`
}
