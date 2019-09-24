FROM alpine:3.8
RUN apk add --no-cache ca-certificates libc6-compat

COPY ./apro-plugin-runner /usr/local/bin/
COPY ./*.opensource.tar.gz /

CMD [ "apro-plugin-runner" ]
