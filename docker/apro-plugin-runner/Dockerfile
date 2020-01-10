FROM alpine:3.8
RUN apk add --no-cache ca-certificates libc6-compat

COPY ./aes-plugin-runner /usr/local/bin/

CMD [ "aes-plugin-runner" ]
