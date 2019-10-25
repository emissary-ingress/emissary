ARG base="frolvlad/alpine-glibc:alpine-3.9"

FROM ${base}
# XXX: this will break if run from an aes build
COPY --from=snapshot-ambassador /buildroot/bin/httptest /usr/local/bin/httptest
ENTRYPOINT ["/usr/local/bin/httptest"]
