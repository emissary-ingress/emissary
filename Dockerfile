FROM golang:1.10-alpine
COPY image .
RUN mkdir /config
ENV USE_STATSD false
ENV RUNTIME_ROOT /
ENV RUNTIME_SUBDIRECTORY config
ENTRYPOINT [ "./ratelimit" ]
