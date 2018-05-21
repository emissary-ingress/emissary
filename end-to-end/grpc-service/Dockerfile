FROM alpine:3.7

WORKDIR /srv

RUN apk add --no-cache \
    build-base \
    gcc \
    python3 \
    python3-dev \
    openssl-dev && \
  python3 -m ensurepip && \
  rm -r /usr/lib/python*/ensurepip && \
  pip3 install --upgrade pip setuptools && \
  if [[ ! -e /usr/bin/pip ]]; then ln -s pip3 /usr/bin/pip; fi && \
  if [[ ! -e /usr/bin/python ]]; then ln -sf /usr/bin/python3 /usr/bin/python; fi && \
  rm -r /root/.cache

COPY requirements.txt .
RUN pip install -Ur requirements.txt

COPY . .
RUN  python -m grpc_tools.protoc \
            --proto_path=. \
            --python_out=. \
            --grpc_python_out=. \
            helloworld.proto \
    && pip install -e . \
    && chmod +x server.py

EXPOSE 50051
ENTRYPOINT ["./server.py"]
