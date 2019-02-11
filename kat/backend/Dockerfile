FROM golang:1.11.5-alpine3.8

WORKDIR $GOPATH/src/github.com/datawire/ambassador/kat/backend/

ENV GO111MODULE="on"
ENV CGO_ENABLED=0

# git
RUN apk add gcc make git openssl && rm /var/cache/apk/*

RUN ls $GOPATH/src/

COPY xds/envoy ./xds/envoy
COPY services ./services
COPY server.go server.go
COPY server.crt server.crt
COPY server.key server.key

# We commit vendor only for speeding up the build. 
COPY vendor ./vendor
COPY go.mod go.mod
COPY go.sum go.sum

RUN go build -mod=vendor -o bin/server

EXPOSE 8080
ENTRYPOINT ["sh", "-c", "./bin/server"]
