FROM golang:1.12.5-alpine3.9

WORKDIR /kat-server

RUN apk --no-cache add git curl

COPY server.crt server.crt
COPY server.key server.key
COPY kat-server kat-server

ENV GRPC_VERBOSITY=debug
ENV GRPC_TRACE=tcp,http,api

EXPOSE 8080
CMD ./kat-server
