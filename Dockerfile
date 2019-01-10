FROM alpine:3.8
RUN apk add ca-certificates && rm -rf /var/cache/apk/*

COPY bin_linux_amd64/ambassador-oauth /bin/
CMD ["/bin/ambassador-oauth"]
EXPOSE 8080
