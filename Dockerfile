FROM golang:1.11.5-alpine3.8

COPY bin/kat-server /usr/local/bin/kat-server

EXPOSE 8080
ENTRYPOINT ["sh", "-c", "./usr/local/bin/kat-server"]
