FROM alpine:3.8
RUN apk add --no-cache ca-certificates python3 libc6-compat curl
COPY kubeapply kubectl loadtest-generator test.sh /usr/local/bin/
COPY 03-ambassador.yaml /opt/
CMD ["test.sh"]
