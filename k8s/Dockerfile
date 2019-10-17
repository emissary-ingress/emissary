ARG base="frolvlad/alpine-glibc:alpine-3.9"

FROM ${base}
COPY --from=snapshot /buildroot/bin/httptest /usr/local/bin/httptest
ENTRYPOINT ["/usr/local/bin/httptest"]
