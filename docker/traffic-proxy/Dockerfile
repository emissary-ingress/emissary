FROM alpine:3.8
RUN apk add --no-cache openssh && \
    ssh-keygen -A && \
    echo -e "ClientAliveInterval 1\nGatewayPorts yes\nPermitEmptyPasswords yes\nPort 8022\nClientAliveCountMax 10\nPermitRootLogin yes\n" >> /etc/ssh/sshd_config && \
    chmod -R g+r /etc/ssh && \
    echo "telepresence::1000:0:Telepresence User:/app:/bin/ash" >> /etc/passwd && \
    chmod g+w /etc/passwd

USER 1000:0
WORKDIR /app
COPY run.sh .
COPY proxy .
CMD ["/app/run.sh"]
