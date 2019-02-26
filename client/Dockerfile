FROM golang:1.11.5

WORKDIR $GOPATH/src/github.com/datawire/kat-backend

ENV GO111MODULE=on
ENV CGO_ENABLED=0

RUN apt-get install git

COPY . .

RUN cd client && go mod download && \
    go install github.com/mitchellh/gox

ENTRYPOINT ["bash", "-c", "./client/entrypoint.sh"]
