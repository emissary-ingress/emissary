FROM alpine:3.8
RUN apk add --no-cache ca-certificates && \
    wget https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/amd64/kubectl -O /bin/kubectl && \
    chmod +x /bin/kubectl

COPY ./consul_connect_integration /opt/ambassador-pro/
ENTRYPOINT [ "/opt/ambassador-pro/consul_connect_integration" ]
