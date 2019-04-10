FROM alpine:3.8
RUN apk add --no-cache ca-certificates python3
COPY max-load test.sh /usr/local/bin/
CMD ["test.sh"]
