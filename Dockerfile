FROM golang:1.10.3-alpine3.8

# install git
RUN apk add git openssl && rm /var/cache/apk/*

RUN go get -u github.com/golang/dep/...

WORKDIR $GOPATH/src/github.com/datawire/ambassador-oauth 
COPY Gopkg.toml Gopkg.lock ./

RUN dep ensure -vendor-only

COPY . $GOPATH/src/github.com/datawire/ambassador-oauth
RUN go install ./cmd/...

CMD ["ambassador-oauth"]

EXPOSE 8080
