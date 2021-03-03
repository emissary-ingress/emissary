FROM docker.io/frolvlad/alpine-glibc:alpine-3.12_glibc-2.32
RUN apk add --no-cache ca-certificates
COPY model-cluster-agentcom /usr/local/bin/
CMD ["model-cluster-agentcom"]
EXPOSE 3000
