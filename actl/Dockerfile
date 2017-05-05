FROM python:3.6-alpine

RUN set -ex \
    && apk add --no-cache --virtual .fetch-deps openssl \
    && wget -O /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(wget -O - https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
    && chmod a+x /usr/bin/kubectl \
    && apk del .fetch-deps

COPY . /app
WORKDIR /app

RUN pip install --no-cache-dir -r requirements.txt

ENV KUBECONFIG=/kubeconfig

ENTRYPOINT ["python", "actl.py"]
