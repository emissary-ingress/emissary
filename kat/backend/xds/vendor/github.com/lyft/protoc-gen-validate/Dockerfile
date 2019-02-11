FROM ubuntu:xenial

ENV GOPATH /go
ENV PATH "${GOPATH}/bin:${PATH}"

COPY ./scripts/build_container.sh /
RUN ./build_container.sh

WORKDIR ${GOPATH}/src/github.com/lyft/protoc-gen-validate
COPY . .

RUN make build

ENTRYPOINT ["make"]
CMD ["quick"]
