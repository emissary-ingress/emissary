#!/hint/sh
RATELIMIT_IMAGE=$(cat docker/amb-sidecar-ratelimit.docker.knaut-push)
PROXY_IMAGE=$(cat docker/traffic-proxy.docker.knaut-push)
SIDECAR_IMAGE=$(cat docker/app-sidecar.docker.knaut-push)
CONSUL_CONNECT_INTEGRATION_IMAGE=$(cat docker/consul_connect_integration.docker.knaut-push)