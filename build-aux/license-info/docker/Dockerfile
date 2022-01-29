# syntax = docker/dockerfile:1.3-labs
### This file generates the list of dependencies used by an npm application

FROM node:10-alpine as dependencies

RUN apk add --update --no-cache jq

RUN npm install -g license-checker@25.0.1

WORKDIR /app
COPY scan.sh customLicenseFormat.json ./

RUN chmod +x scan.sh

WORKDIR /app/build
