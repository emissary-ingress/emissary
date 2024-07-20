package entrypoint

type IRMapping struct {
	Active       bool     `json:"_active"`
	CacheKey     string   `json:"_cache_key"`
	Errored      bool     `json:"_errored"`
	ReferencedBy []string `json:"_referenced_by"`
	RKey         string   `json:"_rkey"`
	Weight       int      `json:"_weight"`
	// Cluster IRCluster `json:"cluster"`
	ClusterKey   string      `json:"cluster_key"`
	DefaultClass string      `json:"default_class"`
	GroupID      string      `json:"group_id"`
	Headers      []IRHeader  `json:"headers"`
	Host         string      `json:"host"`
	Kind         string      `json:"kind"`
	Location     string      `json:"location"`
	Mappings     []IRMapping `json:"mappings"`
	Name         string      `json:"name"`
	Namespace    string      `json:"namespace"`
	Precedence   int         `json:"precedence"`
	Prefix       string      `json:"prefix"`
	// QueryParameters []IRQueryParameter `json:"query_parameters"`
	// RegexRewrite IRRegexRewrite `json:"regex_rewrite"`
	Resolver string `json:"resolver"`
	Rewrite  string `json:"rewrite"`
	// RouteWeight IRRouteWeight `json:"route_weight"`
	Service   string `json:"service"`
	TimeoutMS int    `json:"timeout_ms"`
}

type IRHeader struct {
	Name  string `json:"name"`
	Regex bool   `json:"regex"`
	Value string `json:"value"`
}

type IRGroup struct {
	Active       bool        `json:"_active"`
	CacheKey     string      `json:"_cache_key"`
	DefaultClass string      `json:"default_class"`
	Errored      bool        `json:"_errored"`
	ReferencedBy []string    `json:"_referenced_by"`
	RKey         string      `json:"_rkey"`
	GroupID      string      `json:"group_id"`
	Headers      []IRHeader  `json:"headers"`
	Host         string      `json:"host"`
	Kind         string      `json:"kind"`
	Location     string      `json:"location"`
	Mappings     []IRMapping `json:"mappings"`
	Name         string      `json:"name"`
	Namespace    string      `json:"namespace"`
	Precedence   int         `json:"precedence"`
	Prefix       string      `json:"prefix"`
	// QueryParameters []IRQueryParameter `json:"query_parameters"`
	// RegexRewrite IRRegexRewrite `json:"regex_rewrite"`
	Rewrite string `json:"rewrite"`
	// RouteWeight IRRouteWeight `json:"route_weight"`
	TimeoutMS int `json:"timeout_ms"`
}

type IR struct {
	Groups []IRGroup `json:"groups"`
}
