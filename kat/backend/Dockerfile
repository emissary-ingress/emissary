FROM golang:1.10.3-alpine3.8
COPY server.go server.go
COPY server.crt server.crt
COPY server.key server.key
EXPOSE 8080
ENTRYPOINT ["go", "run", "server.go"]
