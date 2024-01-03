FROM golang:latest as builder

WORKDIR /app

COPY go.mod go.sum ./
COPY api api
COPY cmd/apiext cmd/apiext
COPY pkg pkg
COPY vendor vendor

RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 go build -mod=vendor -o apiext cmd/apiext/main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/apiext /usr/local/bin/apiext
USER 8888:8888 

ENTRYPOINT ["apiext", "emissary-apiext"]