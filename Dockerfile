FROM golang:1.10.3-alpine3.8

# install git
RUN apk add make git openssl && rm /var/cache/apk/*

RUN go get -u github.com/golang/dep/...

WORKDIR $GOPATH/src/github.com/datawire/ambassador-oauth
COPY . ./

RUN make vendor
RUN make install
RUN make test

CMD ["ambassador-oauth"]

EXPOSE 8080
