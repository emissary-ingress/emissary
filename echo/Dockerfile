FROM golang:1.11

ENV PROTOC_VERSION="3.6.1"

WORKDIR $GOPATH/src/github.com/datawire/kat-backend/echo

RUN apt-get update && apt-get install curl unzip -y
RUN curl -OL https://github.com/google/protobuf/releases/download/v$PROTOC_VERSION/protoc-$PROTOC_VERSION-linux-x86_64.zip
RUN unzip protoc-$PROTOC_VERSION-linux-x86_64.zip -d protoc3
RUN mv protoc3/bin/* /usr/local/bin/
RUN mv protoc3/include/* /usr/local/include/
RUN go get github.com/golang/protobuf/protoc-gen-go

COPY echo/echo.proto echo/echo.proto

RUN protoc -I echo/ echo/echo.proto --go_out=plugins=grpc:echo

ENTRYPOINT ["mv", "echo/echo.pb.go", "/echo"]
