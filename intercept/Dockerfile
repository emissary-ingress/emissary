FROM alpine:3.6 as telepresence-proxy
RUN apk add --no-cache openssh && \
    ssh-keygen -A && \
    echo -e "ClientAliveInterval 1\nGatewayPorts yes\nPermitEmptyPasswords yes\nPort 8022\nClientAliveCountMax 10\nPermitRootLogin yes\n" >> /etc/ssh/sshd_config && \
    chmod -R g+r /etc/ssh && \
    echo "telepresence::1000:0:Telepresence User:/app:/bin/ash" >> /etc/passwd && \
    chmod g+w /etc/passwd
USER 1000:0
WORKDIR /app
COPY intercept/proxy_image/run.sh .
COPY image/proxy .
CMD ["/app/run.sh"]

FROM envoyproxy/envoy:28d5f4118d60f828b1453cd8ad25033f2c8e38ab as telepresence-sidecar
WORKDIR /application
COPY intercept/sidecar_image /application/
RUN wget -q https://s3.amazonaws.com/datawire-static-files/ambex/0.1.0/ambex
RUN chmod 755 ambex
COPY image/sidecar /application/
CMD ["/application/entrypoint.sh"]
