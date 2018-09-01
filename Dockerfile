FROM golang:1.10.3-alpine3.8

# install git
RUN apk add git openssl && rm /var/cache/apk/*

RUN mkdir -p $GOPATH/src/app 
ADD . $GOPATH/src/app

WORKDIR $GOPATH/src/app 
RUN go install 

CMD ["app"]

EXPOSE 8080
