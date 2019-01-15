FROM alpine:3.8
RUN apk add --no-cache ca-certificates && mkdir /go

COPY . .
ENV USE_STATSD false
ENV RUNTIME_ROOT /go/config
ENV RUNTIME_SUBDIRECTORY config
ENTRYPOINT [ "./entrypoint.sh" ]
