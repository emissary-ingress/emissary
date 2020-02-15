package main

import (
	"io"
	"text/template"
)

func writeBootstrapADSYAML(w io.Writer, appPort uint32) error {
	t := template.New("bootstrap-ads.yaml")
	template.Must(t.Parse(`# Base config for an ADS management server on 18000, admin port on 19000
admin:
  access_log_path: /dev/null
  address:
    socket_address:
      address: 127.0.0.1
      port_value: 19000
dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
      - envoy_grpc:
          cluster_name: xds_cluster
  cds_config:
    ads: {}
  lds_config:
    ads: {}
node:
  cluster: test-cluster
  id: test-id
static_resources:
  clusters:
  - name: xds_cluster
    connect_timeout: 1s
    hosts:
    - socket_address:
        address: 127.0.0.1
        port_value: 18000
    http2_protocol_options: {}
  - name: tel-proxy-9000
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9000
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9000
  - name: tel-proxy-9001
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9001
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9001
  - name: tel-proxy-9002
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9002
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9002
  - name: tel-proxy-9003
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9003
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9003
  - name: tel-proxy-9004
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9004
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9004
  - name: tel-proxy-9005
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9005
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9005
  - name: tel-proxy-9006
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9006
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9006
  - name: tel-proxy-9007
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9007
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9007
  - name: tel-proxy-9008
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9008
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9008
  - name: tel-proxy-9009
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9009
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9009
  - name: tel-proxy-9010
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9010
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9010
  - name: tel-proxy-9011
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9011
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9011
  - name: tel-proxy-9012
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9012
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9012
  - name: tel-proxy-9013
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9013
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9013
  - name: tel-proxy-9014
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9014
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9014
  - name: tel-proxy-9015
    connect_timeout: 10s
    type: STRICT_DNS
    load_assignment:
      cluster_name: tel-proxy-9015
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: telepresence-proxy
                port_value: 9015

  - name: app
    connect_timeout: 1s
    type: STATIC
    load_assignment:
      cluster_name: app
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: {{ .AppPort }}
`))
	return t.Execute(w, map[string]interface{}{
		"AppPort": appPort,
	})
}

const listenerJSON = `{
  "@type": "/envoy.api.v2.Listener",
  "name": "test-listener",
  "address": {
    "socket_address": {
      "address": "0.0.0.0",
      "port_value": 9900
    }
  },
  "filter_chains": [
    {
      "filters": [
        {
          "name": "envoy.http_connection_manager",
          "config": {
            "stat_prefix": "sidecar",
            "http_filters": [
              {
                "name": "envoy.router"
              }
            ],
            "rds": {
              "route_config_name": "application_route",
              "config_source": {
                "ads": {
                }
              }
            }
          }
        }
      ]
    }
  ]
}
`

const routeJSON = `{
  "@type": "/envoy.api.v2.RouteConfiguration",
  "name": "application_route",
  "virtual_hosts": [
    {
      "name": "all-the-hosts",
      "domains": [
        "*"
      ],
      "routes": [
        {
          "match": {
            "prefix": "/"
          },
          "route": {
            "cluster": "app"
          }
        }
      ]
    }
  ]
}
`
