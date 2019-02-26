#!/usr/bin/env bash

cd client && gox -osarch="darwin/amd64 linux/amd64" && \
mv client_darwin_amd64 client_linux_amd64 /usr/local/tmp/
