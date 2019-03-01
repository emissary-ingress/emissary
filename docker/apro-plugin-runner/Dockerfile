FROM alpine:3.8
RUN apk add --no-cache ca-certificates bash libc6-compat

COPY ./apro-plugin-runner /usr/local/bin/

CMD [ "apro-plugin-runner" ]
