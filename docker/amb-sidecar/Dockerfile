FROM alpine:3.8
RUN apk add --no-cache ca-certificates bash

# Files that do NOT have the version number embedded
COPY ./run.sh ./ratelimit /opt/ambassador-pro/

# Files that DO have the version number embedded
COPY ./ambassador-oauth ./apictl /opt/ambassador-pro/

CMD [ "/opt/ambassador-pro/run.sh" ]
EXPOSE 8080
