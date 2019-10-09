FROM alpine:3.8
RUN apk add --no-cache ca-certificates libc6-compat

COPY ./kubectl /usr/local/bin/
COPY ./consul_connect_integration /opt/ambassador-pro/
COPY ./*.opensource.tar.gz /

ENTRYPOINT [ "/opt/ambassador-pro/consul_connect_integration" ]
