# The `test-http` image gets built by `pkg/kubeapply` for use by
# various kubeapply-templated YAML files.

FROM golang:1.15
COPY httptest.go /usr/local/bin/httptest.go
RUN go build -o /usr/local/bin/httptest /usr/local/bin/httptest.go
ENTRYPOINT ["/usr/local/bin/httptest"]
