FROM alpine:3.8

ARG KUBECTL_VERSION=v1.13.0

RUN apk add --no-cache ca-certificates \
    && wget https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl -O /bin/kubectl \
    && chmod +x /bin/kubectl

COPY . .
ENTRYPOINT [ "./consul_connect_integration" ]
