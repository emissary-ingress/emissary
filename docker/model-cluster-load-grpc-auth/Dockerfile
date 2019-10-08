FROM alpine:3.8
RUN apk add --no-cache ca-certificates libc6-compat
COPY model-cluster-load-grpc-auth /usr/local/bin/
CMD ["model-cluster-load-grpc-auth"]
EXPOSE 3000
