#!/bin/sh

if [ -z "${APPPORT}" ]; then
    echo "ERROR: APPPORT env var not configured."
    echo "(I don't know what port your app uses)"
    echo "Please set APPPORT in your k8s manifest."
    sleep 86400
    exit 1
fi

cat > data/cluster-app.json <<EOF
{
  "@type": "/envoy.api.v2.Cluster",
  "name": "app",
  "connect_timeout": "1s",
  "type": "STATIC",
  "load_assignment": {
    "cluster_name": "app",
    "endpoints": [
      {
        "lb_endpoints": [
          {
            "endpoint": {
              "address": {
                "socket_address": {
                  "address": "127.0.0.1",
                  "port_value": ${APPPORT}
                }
              }
            }
          }
        ]
      }
    ]
  }
}
EOF

./ambex -watch data > /dev/null 2>&1 &
./sidecar &
envoy -l debug -c bootstrap-ads.yaml
