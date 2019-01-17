#!/hint/sh
RATELIMIT_IMAGE=$(cat docker/amb-sidecar-ratelimit.docker.knaut-push)
PROXY_IMAGE=$(cat docker/traffic-proxy.docker.knaut-push)
SIDECAR_IMAGE=$(cat docker/app-sidecar.docker.knaut-push)
