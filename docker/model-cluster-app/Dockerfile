FROM alpine:3.8
RUN apk add --no-cache ca-certificates libc6-compat
WORKDIR /app
COPY model-cluster-app /app/
CMD ["/app/model-cluster-app"]
