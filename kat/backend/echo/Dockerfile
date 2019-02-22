FROM golang:1.11

ENV PROTOC_VERSION="3.6.1"
ENV SRC_PATH="github.com/datawire/ambassador/kat/backend/echo"

WORKDIR $GOPATH/src/$SRC_PATH

RUN apt-get update && apt-get install curl unzip -y

RUN curl -OL https://github.com/google/protobuf/releases/download/v$PROTOC_VERSION/protoc-$PROTOC_VERSION-linux-x86_64.zip
RUN unzip protoc-$PROTOC_VERSION-linux-x86_64.zip -d protoc3
RUN mv protoc3/bin/* /usr/local/bin/
RUN mv protoc3/include/* /usr/local/include/

RUN go get -u github.com/golang/protobuf/protoc-gen-go

COPY echo-service.proto echo-service/echo-service.proto

RUN protoc -I echo-service/ echo-service/echo-service.proto --go_out=plugins=grpc:echo-service

ENTRYPOINT ["mv", "echo-service/echo-service.pb.go", "/echo"]
