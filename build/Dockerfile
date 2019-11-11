ARG AES_GOVERSION
FROM golang:${AES_GOVERSION}

ARG CUR_DIR
ARG UID

RUN apt update
RUN apt install rsync -y

RUN mkdir -p ${CUR_DIR}
RUN mkdir -p /mnt/goproxy

# USER ${UID}


# ENV GOCACHE=/tmp/go-cache
# ENV GOPROXY=file:///mnt/goproxy
# ENV GOPATH=/home/circleci/.go_workspace:/usr/local/go_workspace
# ENV GOOS=linux
# ENV GOARCH=amd64
# ENV CGO_ENABLED=1
# ENV GO111MODULE=on

LABEL product AES
LABEL component plugin-builder

WORKDIR ${CUR_DIR}

ENTRYPOINT [ "tail", "-f", "/dev/null" ]