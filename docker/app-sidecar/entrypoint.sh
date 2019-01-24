#!/bin/bash

if [ -z "${APPPORT}" ]; then
    echo "ERROR: APPPORT env var not configured."
    echo "(I don't know what port your app uses)"
    echo "Please set APPPORT in your k8s manifest."
    sleep 86400
    exit 1
fi

cat >> bootstrap-ads.yaml <<EOF
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
                port_value: ${APPPORT}
EOF

# Initialize job management
trap 'jobs -p | xargs -r kill --' INT
launch() {
	(
		trap 'echo "Exited with $?: $*"' EXIT
		env "$@"
	) &
}

# Launch each of the worker processes
launch sh -c 'exec ./ambex -watch data >/dev/null 2>&1'
launch ./app-sidecar
launch envoy -l debug -c bootstrap-ads.yaml

# Wait for one of them to quit, then kill the others
wait -n
r=$?
echo ' ==> One of the worker processes exited; shutting down the others <=='
while test -n "$(jobs -p)"; do
	jobs -p | xargs -r kill --
	wait -n
done
echo 'Finished shutting down'
exit $r
