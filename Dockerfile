FROM golang:1.10.3-alpine3.8

# install git
RUN apk add git openssl && rm /var/cache/apk/*

WORKDIR /go/src
ADD . /go/src

RUN go get -d

CMD ["go", "run", "*.go"]

EXPOSE 8080
