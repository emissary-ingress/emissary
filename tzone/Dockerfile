FROM quay.io/datawire/ambassador-envoy-alpine-stripped:v1.9.0-619-g5830eaa1d
RUN apk add iptables curl bind-tools
WORKDIR /work
COPY teleproxy kat-server server.crt server.key ./
COPY run-stuff.sh routes.json load-routes.sh ./
