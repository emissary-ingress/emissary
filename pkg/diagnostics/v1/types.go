package diagnostics

const ApiVersion = "v1"
const ContentTypeJSON = "application/json"

type Diagnostics struct {
	ActiveElements      []string               `json:"active_elements"`
	AmbassadorElements  *AmbassadorElements    `json:"ambassador_elements"`
	AmbassadorResolvers []*AmbassadorResolver  `json:"ambassador_resolvers"`
	AmbassadorResources *AmbassadorResources   `json:"ambassador_resources"`
	AmbassadorServices  []*AmbassadorService   `json:"ambassador_services"`
	BannerContent       string                 `json:"banner_content"`
	ClusterInfo         *ClusterInfo           `json:"cluster_info"`
	ClusterStats        *ClusterStats          `json:"cluster_stats"`
	EnvoyElements       *EnvoyElements         `json:"envoy_elements"`
	EnvoyResources      *EnvoyResources        `json:"envoy_resources"`
	EnvoyStatus         *EnvoyStatus           `json:"envoy_status"`
	Errors              [][]string             `json:"errors"`
	Groups              *Groups                `json:"groups"`
	Loginfo             *Loginfo               `json:"loginfo"`
	Notices             []*Notice              `json:"notices"`
	RouteInfo           []*RouteInfo           `json:"route_info"`
	SourceMap           map[string]interface{} `json:"source_map"`
	System              *System                `json:"system"`
	Tlscontexts         []*TLSContext          `json:"tlscontexts"`
}

type AmbassadorElements map[string]struct {
	Kind          string `json:"kind"`
	Location      string `json:"location"`
	Parent        string `json:"parent"`
	Serialization string `json:"serialization"`
}

type AmbassadorResolver struct {
	Source string   `json:"_source"`
	Groups []string `json:"groups"`
	Kind   string   `json:"kind"`
	Name   string   `json:"name"`
}

type AmbassadorResources struct{}

type AmbassadorService struct {
	ServiceWeight float64 `json:"_service_weight"`
	Source        string  `json:"_source"`
	Cluster       string  `json:"cluster"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
}

type ClusterInfo map[string]struct {
	Active               bool     `json:"_active"`
	CacheKey             string   `json:"_cache_key"`
	Errored              bool     `json:"_errored"`
	Hcolor               string   `json:"_hcolor"`
	Health               string   `json:"_health"`
	Hmetric              string   `json:"_hmetric"`
	Hostname             string   `json:"_hostname"`
	IsSidecar            bool     `json:"_is_sidecar"`
	Namespace_           string   `json:"_namespace"`
	Port                 int      `json:"_port"`
	ReferencedBy         []string `json:"_referenced_by"`
	Resolver             string   `json:"_resolver"`
	Rkey                 string   `json:"_rkey"`
	ClusterIdleTimeoutMs int      `json:"cluster_idle_timeout_ms"`
	ConnectTimeoutMs     int      `json:"connect_timeout_ms"`
	EnableEndpoints      bool     `json:"enable_endpoints"`
	EnableIpv4           bool     `json:"enable_ipv4"`
	EnableIpv6           bool     `json:"enable_ipv6"`
	EnvoyName            string   `json:"envoy_name"`
	HostRewrite          string   `json:"host_rewrite"`
	IgnoreCluster        bool     `json:"ignore_cluster"`
	Kind                 string   `json:"kind"`
	LbType               string   `json:"lb_type"`
	Location             string   `json:"location"`
	Name                 string   `json:"name"`
	Namespace            string   `json:"namespace"`
	RespectDNSTTL        bool     `json:"respect_dns_ttl"`
	Service              string   `json:"service"`
	StatsName            string   `json:"stats_name"`
	Targets              []struct {
		IP         string `json:"ip"`
		Port       int    `json:"port"`
		TargetKind string `json:"target_kind"`
	} `json:"targets"`
	TLSContext struct {
		Active            bool     `json:"_active"`
		AmbassadorEnabled bool     `json:"_ambassador_enabled"`
		Errored           bool     `json:"_errored"`
		Rkey              string   `json:"_rkey"`
		IsFallback        bool     `json:"is_fallback"`
		Kind              string   `json:"kind"`
		Location          string   `json:"location"`
		Name              string   `json:"name"`
		Namespace         string   `json:"namespace"`
		SecretInfo        struct{} `json:"secret_info"`
	} `json:"tls_context"`
	Type   string   `json:"type"`
	Urls   []string `json:"urls"`
	Weight int      `json:"weight"`
}

type ClusterStats map[string]struct {
	Hcolor  string `json:"hcolor"`
	Health  string `json:"health"`
	Hmetric string `json:"hmetric"`
	Reason  string `json:"reason"`
	Valid   bool   `json:"valid"`
}

type EnvoyElementsCluster struct {
	AltStatName               string `json:"alt_stat_name"`
	CommonHTTPProtocolOptions struct {
		IdleTimeout string `json:"idle_timeout"`
	} `json:"common_http_protocol_options"`
	ConnectTimeout       string `json:"connect_timeout"`
	DNSLookupFamily      string `json:"dns_lookup_family"`
	LbPolicy             string `json:"lb_policy"`
	HTTP2ProtocolOptions struct {
	} `json:"http2_protocol_options"`
	LoadAssignment struct {
		ClusterName string `json:"cluster_name"`
		Endpoints   []struct {
			LbEndpoints []struct {
				Endpoint struct {
					Address struct {
						SocketAddress struct {
							Address   string `json:"address"`
							PortValue int    `json:"port_value"`
							Protocol  string `json:"protocol"`
						} `json:"socket_address"`
					} `json:"address"`
				} `json:"endpoint"`
			} `json:"lb_endpoints"`
		} `json:"endpoints"`
	} `json:"load_assignment"`
	TransportSocket struct {
		Name        string `json:"name"`
		TypedConfig struct {
			Type             string `json:"@type"`
			CommonTLSContext struct {
			} `json:"common_tls_context"`
		} `json:"typed_config"`
	} `json:"transport_socket"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type EnvoyElementsRoute struct {
	Match struct {
		CaseSensitive bool `json:"case_sensitive"`
		Headers       []struct {
			ExactMatch string `json:"exact_match"`
			Name       string `json:"name"`
		} `json:"headers"`
		Prefix          string `json:"prefix"`
		RuntimeFraction struct {
			DefaultValue struct {
				Denominator string `json:"denominator"`
				Numerator   int    `json:"numerator"`
			} `json:"default_value"`
			RuntimeKey string `json:"runtime_key"`
		} `json:"runtime_fraction"`
	} `json:"match"`
	Route struct {
		Cluster            string      `json:"cluster"`
		IdleTimeout        string      `json:"idle_timeout"`
		HostRewriteLiteral string      `json:"host_rewrite_literal"`
		PrefixRewrite      string      `json:"prefix_rewrite"`
		Priority           interface{} `json:"priority"`
		Timeout            string      `json:"timeout"`
	} `json:"route"`
}

type EnvoyElements map[string]struct {
	Cluster []*EnvoyElementsCluster `json:"cluster"`
	Route   []*EnvoyElementsRoute   `json:"route"`
}
type EnvoyResources interface{}
type EnvoyStatus struct {
	Alive       bool   `json:"alive"`
	Ready       bool   `json:"ready"`
	SinceUpdate string `json:"since_update"`
	Uptime      string `json:"uptime"` // OK
}

type Groups map[string]struct {
	Active       bool     `json:"_active"`
	CacheKey     string   `json:"_cache_key"`
	Errored      bool     `json:"_errored"`
	ReferencedBy []string `json:"_referenced_by"`
	Rkey         string   `json:"_rkey"`
	Cors         struct {
		AllowCredentials       bool `json:"allow_credentials"`
		AllowOriginStringMatch []struct {
			Exact string `json:"exact"`
		} `json:"allow_origin_string_match"`
		FilterEnabled struct {
			DefaultValue struct {
				Denominator string `json:"denominator"`
				Numerator   int    `json:"numerator"`
			} `json:"default_value"`
			RuntimeKey string `json:"runtime_key"`
		} `json:"filter_enabled"`
	} `json:"cors"`
	ClusterIdleTimeoutMs int           `json:"cluster_idle_timeout_ms"`
	DefaultClass         string        `json:"default_class"`
	GroupID              string        `json:"group_id"`
	GroupWeight          []interface{} `json:"group_weight"`
	GRPC                 bool          `json:"grpc"`
	Headers              []struct {
		Name  string `json:"name"`
		Regex bool   `json:"regex"`
		Value string `json:"value"`
	} `json:"headers"`
	Host        string `json:"host"`
	HostRewrite string `json:"host_rewrite"`
	Kind        string `json:"kind"`
	Location    string `json:"location"`
	Mappings    []struct {
		Active         bool   `json:"_active"`
		Errored        bool   `json:"_errored"`
		Rkey           string `json:"_rkey"`
		ClusterName    string `json:"cluster_name"`
		ClusterService string `json:"cluster_service"`
		Host           string `json:"host"`
		Location       string `json:"location"`
		Name           string `json:"name"`
		Prefix         string `json:"prefix"`
		Rewrite        string `json:"rewrite"`
	} `json:"mappings"`
	MetadataLabels struct {
		AmbassadorCrd           string `json:"ambassador_crd"`
		AppKubernetesIoInstance string `json:"app.kubernetes.io/instance"`
	} `json:"metadata_labels"`
	Name            string        `json:"name"`
	Namespace       string        `json:"namespace"`
	Precedence      int           `json:"precedence"`
	Prefix          string        `json:"prefix"`
	QueryParameters []interface{} `json:"query_parameters"`
	RegexRewrite    struct{}      `json:"regex_rewrite"`
	Rewrite         string        `json:"rewrite"`
	TimeoutMs       int           `json:"timeout_ms"`
	Serialization   string        `json:"serialization"`
}

type Loginfo struct {
	All string `json:"all"`
}

type RouteInfo struct {
	GroupID string `json:"_group_id"`
	Route   struct {
		Active       bool          `json:"_active"`
		CacheKey     string        `json:"_cache_key"`
		Errored      bool          `json:"_errored"`
		ReferencedBy []string      `json:"_referenced_by"`
		Rkey         string        `json:"_rkey"`
		DefaultClass string        `json:"default_class"`
		GroupID      string        `json:"group_id"`
		GroupWeight  []interface{} `json:"group_weight"`
		Headers      []struct {
			Name  string `json:"name"`
			Regex bool   `json:"regex"`
			Value string `json:"value"`
		} `json:"headers"`
		Host     string `json:"host"`
		Kind     string `json:"kind"`
		Location string `json:"location"`
		Mappings []struct {
			Active            bool   `json:"_active"`
			CacheKey          string `json:"_cache_key"`
			Errored           bool   `json:"_errored"`
			Rkey              string `json:"_rkey"`
			Weight            int    `json:"_weight"`
			AddRequestHeaders struct {
			} `json:"add_request_headers"`
			AddResponseHeaders struct {
			} `json:"add_response_headers"`
			Cluster struct {
				Active           bool     `json:"_active"`
				CacheKey         string   `json:"_cache_key"`
				Errored          bool     `json:"_errored"`
				Hostname         string   `json:"_hostname"`
				IsSidecar        bool     `json:"_is_sidecar"`
				Namespace        string   `json:"_namespace"`
				Port             int      `json:"_port"`
				ReferencedBy     []string `json:"_referenced_by"`
				Resolver         string   `json:"_resolver"`
				Rkey             string   `json:"_rkey"`
				ConnectTimeoutMs int      `json:"connect_timeout_ms"`
				EnableEndpoints  bool     `json:"enable_endpoints"`
				EnableIpv4       bool     `json:"enable_ipv4"`
				EnableIpv6       bool     `json:"enable_ipv6"`
				EnvoyName        string   `json:"envoy_name"`
				IgnoreCluster    bool     `json:"ignore_cluster"`
				Kind             string   `json:"kind"`
				LbType           string   `json:"lb_type"`
				Location         string   `json:"location"`
				Name             string   `json:"name"`
				Namespace0       string   `json:"namespace"`
				RespectDNSTTL    bool     `json:"respect_dns_ttl"`
				Service          string   `json:"service"`
				StatsName        string   `json:"stats_name"`
				Targets          []struct {
					IP         string `json:"ip"`
					Port       int    `json:"port"`
					TargetKind string `json:"target_kind"`
				} `json:"targets"`
				Type string   `json:"type"`
				Urls []string `json:"urls"`
			} `json:"cluster"`
			ClusterKey   string `json:"cluster_key"`
			DefaultClass string `json:"default_class"`
			GroupID      string `json:"group_id"`
			Headers      []struct {
				Name  string `json:"name"`
				Regex bool   `json:"regex"`
				Value string `json:"value"`
			} `json:"headers"`
			Host           string `json:"host"`
			Kind           string `json:"kind"`
			Location       string `json:"location"`
			MetadataLabels struct {
				AmbassadorCrd           string `json:"ambassador_crd"`
				AppKubernetesIoInstance string `json:"app.kubernetes.io/instance"`
			} `json:"metadata_labels"`
			Name            string        `json:"name"`
			Namespace       string        `json:"namespace"`
			Precedence      int           `json:"precedence"`
			Prefix          string        `json:"prefix"`
			QueryParameters []interface{} `json:"query_parameters"`
			RegexRewrite    struct {
			} `json:"regex_rewrite"`
			Resolver      string        `json:"resolver"`
			Rewrite       string        `json:"rewrite"`
			RouteWeight   []interface{} `json:"route_weight"`
			Serialization string        `json:"serialization"`
			Service       string        `json:"service"`
		} `json:"mappings"`
		MetadataLabels struct {
			AmbassadorCrd           string `json:"ambassador_crd"`
			AppKubernetesIoInstance string `json:"app.kubernetes.io/instance"`
		} `json:"metadata_labels"`
		Name            string        `json:"name"`
		Namespace       string        `json:"namespace"`
		Precedence      int           `json:"precedence"`
		Prefix          string        `json:"prefix"`
		QueryParameters []interface{} `json:"query_parameters"`
		RegexRewrite    struct {
		} `json:"regex_rewrite"`
		Rewrite       string `json:"rewrite"`
		Serialization string `json:"serialization"`
	} `json:"_route"`
	Source   string `json:"_source"`
	Clusters []struct {
		Active           bool     `json:"_active"`
		CacheKey         string   `json:"_cache_key"`
		Errored          bool     `json:"_errored"`
		Hcolor           string   `json:"_hcolor"`
		Health           string   `json:"_health"`
		Hmetric          string   `json:"_hmetric"`
		Hostname         string   `json:"_hostname"`
		IsSidecar        bool     `json:"_is_sidecar"`
		Namespace        string   `json:"_namespace"`
		Port             int      `json:"_port"`
		ReferencedBy     []string `json:"_referenced_by"`
		Resolver         string   `json:"_resolver"`
		Rkey             string   `json:"_rkey"`
		ConnectTimeoutMs int      `json:"connect_timeout_ms"`
		EnableEndpoints  bool     `json:"enable_endpoints"`
		EnableIpv4       bool     `json:"enable_ipv4"`
		EnableIpv6       bool     `json:"enable_ipv6"`
		EnvoyName        string   `json:"envoy_name"`
		IgnoreCluster    bool     `json:"ignore_cluster"`
		Kind             string   `json:"kind"`
		LbType           string   `json:"lb_type"`
		Location         string   `json:"location"`
		Name             string   `json:"name"`
		Namespace0       string   `json:"namespace"`
		RespectDNSTTL    bool     `json:"respect_dns_ttl"`
		Service          string   `json:"service"`
		StatsName        string   `json:"stats_name"`
		Targets          []struct {
			IP         string `json:"ip"`
			Port       int    `json:"port"`
			TargetKind string `json:"target_kind"`
		} `json:"targets"`
		Type   string   `json:"type"`
		Urls   []string `json:"urls"`
		Weight int      `json:"weight"`
	} `json:"clusters"`
	Headers    []interface{} `json:"headers"`
	Host       string        `json:"host"`
	Key        string        `json:"key"`
	Method     string        `json:"method"`
	Precedence int           `json:"precedence"`
	Prefix     string        `json:"prefix"`
	Rewrite    string        `json:"rewrite"`
}

type Notice struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type System struct {
	AmbassadorID        string        `json:"ambassador_id"`
	AmbassadorNamespace string        `json:"ambassador_namespace"`
	BootTime            string        `json:"boot_time"`
	ClusterID           string        `json:"cluster_id"`
	DebugMode           bool          `json:"debug_mode"`
	EndpointsEnabled    bool          `json:"endpoints_enabled"`
	EnvFailures         []interface{} `json:"env_failures"`
	EnvGood             bool          `json:"env_good"`
	EnvStatus           struct {
		ErrorCheck struct {
			Specifics [][]interface{} `json:"specifics"`
			Status    bool            `json:"status"`
		} `json:"Error check"`
		Mappings struct {
			Specifics [][]interface{} `json:"specifics"`
			Status    bool            `json:"status"`
		} `json:"Mappings"`
		TLS struct {
			Specifics [][]interface{} `json:"specifics"`
			Status    bool            `json:"status"`
		} `json:"TLS"`
	} `json:"env_status"`
	Hostname        string `json:"hostname"`
	HrUptime        string `json:"hr_uptime"`
	KnativeEnabled  bool   `json:"knative_enabled"`
	LatestSnapshot  string `json:"latest_snapshot"`
	SingleNamespace bool   `json:"single_namespace"`
	StatsdEnabled   bool   `json:"statsd_enabled"`
	Version         string `json:"version"`
}
type TLSContext struct {
	Active       bool     `json:"_active"`
	Errored      bool     `json:"_errored"`
	ReferencedBy []string `json:"_referenced_by"`
	Rkey         string   `json:"_rkey"`
	Hosts        []string `json:"hosts"`
	IsFallback   bool     `json:"is_fallback"`
	Kind         string   `json:"kind"`
	Location     string   `json:"location"`
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	SecretInfo   struct {
		CertChainFile  string `json:"cert_chain_file"`
		PrivateKeyFile string `json:"private_key_file"`
		Secret         string `json:"secret"`
	} `json:"secret_info"`
}
