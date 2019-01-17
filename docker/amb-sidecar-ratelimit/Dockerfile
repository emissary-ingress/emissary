FROM alpine:3.8
RUN apk add --no-cache ca-certificates bash

COPY . .
ENTRYPOINT [ "./entrypoint.sh" ]
