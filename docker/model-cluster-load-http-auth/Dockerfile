FROM alpine:3.8
RUN apk add --no-cache ca-certificates libc6-compat
COPY model-cluster-load-http-auth /usr/local/bin/
CMD ["model-cluster-load-http-auth"]
EXPOSE 3000
