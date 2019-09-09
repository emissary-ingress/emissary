FROM golang:1.12.5-alpine3.9

WORKDIR $GOPATH/src/github.com/datawire/kat-backend

ENV GO111MODULE=on
ENV CGO_ENABLED=0

RUN apk add git curl && rm /var/cache/apk/*

COPY server.crt server.crt
COPY server.key server.key
COPY go.mod go.mod
COPY go.sum go.sum
COPY server.go server.go
COPY echo/ echo/
COPY services/ services/

RUN go build -o bin/kat-server

EXPOSE 8080
ENTRYPOINT ["sh", "-c", "GRPC_VERBOSITY=debug GRPC_TRACE=tcp,http,api ./bin/kat-server"]
