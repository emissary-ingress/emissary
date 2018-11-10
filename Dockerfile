FROM golang:1.10 as builder
RUN curl https://glide.sh/get | sh
COPY . .
RUN CGO_ENABLED=0 make compile

FROM golang:1.10-alpine
COPY --from=builder /go/bin/ratelimit .
RUN mkdir /config
ENV USE_STATSD false
ENV RUNTIME_ROOT /
ENV RUNTIME_SUBDIRECTORY config
ENTRYPOINT [ "./ratelimit" ]
