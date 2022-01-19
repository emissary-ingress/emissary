# syntax = docker/dockerfile:1.3-labs
### This file generates the list of dependencies used by an npm application

FROM node:10-alpine as dependencies

WORKDIR /app
ADD . .

RUN <<EOF sh -ex
    mkdir /dependencies
    npm install
    npx license-checker --json --unknown | tee /dependencies/dependencies.json
    ls -al /dependencies/dependencies.json
EOF

FROM scratch AS export-stage
COPY --from=dependencies /dependencies/dependencies.json .