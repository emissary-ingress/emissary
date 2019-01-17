#FROM golang:1.11-alpine3.8 as foundation
#
#ARG KUBECTL_VERSION=v1.13.0
#ENV GO111MODULE=on
#
#WORKDIR /build
#RUN apk add --no-cache \
#        ca-certificates \
#        git \
#    && wget https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl -O /bin/kubectl \
#    && chmod +x /bin/kubectl
#
#COPY go.mod .
#COPY go.sum .
#
#RUN go mod download
#
#FROM foundation as builder
#
#COPY . .
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
#    go build -o bin/ambpro-consul-connector -a -tags netgo -ldflags '-s -w "-extldflags=-static"'
#
#FROM alpine:3.8 as runtime
#
#COPY --from=builder /bin/kubectl /bin/kubectl
#COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
#COPY --from=builder /build/bin/ambpro-consul-connector /bin/ambpro-consul-connector
#
#ENTRYPOINT ["/bin/ambpro-consul-connector"]

FROM alpine:3.8
RUN apk add --no-cache ca-certificates

COPY . .
ENTRYPOINT [ "./consul_connect_integration" ]
