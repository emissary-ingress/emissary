FROM alpine:3.7

RUN apk --update add socat

CMD ["socat", "-d", "UDP-RECVFROM:8125,fork", "UDP-SENDTO:statsd-sink:8125"]
