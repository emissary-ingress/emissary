#!/bin/bash

set -e

# Base dependencies.
apt-get update
apt-get install --no-install-recommends -y unzip wget git ca-certificates make software-properties-common

# go 1.8
add-apt-repository ppa:longsleep/golang-backports
apt-get update
apt-get install --no-install-recommends -y golang-1.8 golang-1.8-race-detector-runtime
mkdir -p "${GOPATH}"
ln -s /usr/lib/go-1.8/bin/go /usr/local/bin

# protoc
PROTOC_VER=3.4.0
PROTOC_REL=protoc-"${PROTOC_VER}"-linux-x86_64.zip
pushd /tmp
wget https://github.com/google/protobuf/releases/download/v"${PROTOC_VER}/${PROTOC_REL}"
unzip "${PROTOC_REL}" -d protoc
mv protoc /usr/local
ln -s /usr/local/protoc/bin/protoc /usr/local/bin
popd

# protoc-gen-go
PROTOC_GEN_GO=github.com/golang/protobuf/protoc-gen-go
PROTOC_GEN_GO_PATH="$GOPATH/src/$PROTOC_GEN_GO"
PROTOC_GEN_GO_VER=c9c7427a2a70d2eb3bafa0ab2dc163e45f143317
go get -u "$PROTOC_GEN_GO"
pushd "${PROTOC_GEN_GO_PATH}"
git checkout "$PROTOC_GEN_GO_VER"
go install
popd

# Bazel
apt-get install --no-install-recommends -y openjdk-8-jdk curl
echo "deb [arch=amd64] http://storage.googleapis.com/bazel-apt stable jdk1.8" | tee /etc/apt/sources.list.d/bazel.list
curl https://bazel.build/bazel-release.pub.gpg | apt-key add -
apt-get update
apt-get install --no-install-recommends -y bazel

# Cleanup
apt-get clean
rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
