FROM golang:1.11

ENV PROTOC_VERSION="3.5.1"
ENV PROTOC_WEB_VERSION="1.0.3"

WORKDIR $GOPATH/src/github.com/datawire/kat-backend/echo

RUN apt-get update && apt-get install curl unzip -y
RUN go get github.com/golang/protobuf/protoc-gen-go

# Installs protoc.
RUN curl -OL https://github.com/google/protobuf/releases/download/v$PROTOC_VERSION/protoc-$PROTOC_VERSION-linux-x86_64.zip
RUN unzip protoc-$PROTOC_VERSION-linux-x86_64.zip -d protoc3
RUN mv protoc3/bin/* /usr/local/bin/
RUN mv protoc3/include/* /usr/local/include/

# Installs protoc-web.
RUN curl -OL https://github.com/grpc/grpc-web/releases/download/$PROTOC_WEB_VERSION/protoc-gen-grpc-web-$PROTOC_WEB_VERSION-linux-x86_64
RUN mv protoc-gen-grpc-web-$PROTOC_WEB_VERSION-linux-x86_64 /usr/local/bin/protoc-gen-grpc-web
RUN chmod +x /usr/local/bin/protoc-gen-grpc-web

COPY echo/echo.proto echo/echo.proto

RUN protoc -I echo/ echo/echo.proto --go_out=plugins=grpc:echo

RUN protoc -I echo/ echo/echo.proto --grpc-web_out=import_style=commonjs,mode=grpcwebtext:echo

RUN protoc -I echo/ echo/echo.proto --js_out=import_style=commonjs:echo

RUN ls echo/ 

ENTRYPOINT ["mv", "echo/echo.pb.go", "echo/echo_grpc_web_pb.js", "echo/echo_pb.js", "/echo"]
