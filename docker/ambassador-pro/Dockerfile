FROM alpine:3.8
RUN apk add ca-certificates && rm -rf /var/cache/apk/*

COPY ./ambassador-oauth /bin/
CMD ["/bin/ambassador-oauth"]
EXPOSE 8080
