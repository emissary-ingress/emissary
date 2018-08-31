FROM golang:1.10.3-alpine3.8

# install git
RUN apk add git openssl && rm /var/cache/apk/*

WORKDIR /go/src
ADD . /go/src

CMD ["go", "run", "controller.go", "main.go"]

EXPOSE 8080
