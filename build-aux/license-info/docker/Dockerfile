# syntax = docker/dockerfile:1.3-labs
### This file generates the list of dependencies used by an npm application

FROM node:10-alpine as dependencies

WORKDIR /js_scanner
RUN echo '{"name": "", "version": "", "description": "", "licenseFile": "", "licenseText": "", "licenseModified": ""}' > ./customLicenseFormat.json

WORKDIR /app
ADD . .

RUN apk add --update --no-cache jq

RUN mkdir /dependencies && \
    npm install && \
    npm install -g license-checker@25.0.1


RUN export THIS_PACKAGE=$(cat package.json | jq -r '.name + "@" + .version') && \
    license-checker --excludePackages "$THIS_PACKAGE" --customPath /js_scanner/customLicenseFormat.json --json > /dependencies/dependencies.json

FROM scratch AS export-stage
COPY --from=dependencies /dependencies/dependencies.json .