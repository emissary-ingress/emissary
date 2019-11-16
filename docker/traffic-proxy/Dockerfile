FROM alpine:3.8
RUN apk add --no-cache openssh ca-certificates libc6-compat && \
    ssh-keygen -A && \
    echo -e "ClientAliveInterval 1\nGatewayPorts yes\nPermitEmptyPasswords yes\nPort 8022\nClientAliveCountMax 10\nPermitRootLogin yes\n" >> /etc/ssh/sshd_config && \
    chmod -R g+r /etc/ssh && \
    echo "telepresence::1000:0:Telepresence User:/app:/bin/ash" >> /etc/passwd && \
    chmod g+w /etc/passwd

COPY ./run.sh ./traffic-proxy /app/

USER 1000:0
WORKDIR /app
CMD ["/app/run.sh"]
