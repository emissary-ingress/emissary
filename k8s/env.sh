#!/hint/sh
RATELIMIT_IMAGE=$(cat docker/ambassador-ratelimit.docker.knaut-push)
PROXY_IMAGE=$(cat docker/traffic-proxy.docker.knaut-push)
SIDECAR_IMAGE=$(cat docker/traffic-sidecar.docker.knaut-push)
