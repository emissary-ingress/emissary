FROM alpine:3.8
RUN apk add --no-cache ca-certificates libc6-compat
COPY model-cluster-http-auth /usr/local/bin/
CMD ["model-cluster-http-auth"]
EXPOSE 4000
