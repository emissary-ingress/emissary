FROM golang:1.10.3-alpine3.8

# install git
RUN apk add make git openssl && rm /var/cache/apk/*

RUN go get -u github.com/golang/dep/...

WORKDIR /root/ambassador-oauth
COPY . ./
RUN ln -s ../../../.. .go-workspace/src/github.com/datawire/ambassador-oauth


RUN make go-build

CMD ["./bin_linux_amd64/ambassador-oauth"]

EXPOSE 8080
