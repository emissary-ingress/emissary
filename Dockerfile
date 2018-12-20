FROM golang:1.10-alpine
COPY image .
COPY entrypoint.sh .
ENV USE_STATSD false
ENV RUNTIME_ROOT /go/config
ENV RUNTIME_SUBDIRECTORY config
ENTRYPOINT [ "./entrypoint.sh" ]
