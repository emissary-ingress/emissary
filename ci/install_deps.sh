#!/bin/bash

wget https://get.helm.sh/helm-v2.16.1-linux-amd64.tar.gz -O /tmp/helm2.tar.gz
wget https://get.helm.sh/helm-v3.0.1-linux-amd64.tar.gz -O /tmp/helm.tar.gz

tar -xzf /tmp/helm2.tar.gz
sudo mv linux-amd64/helm /usr/local/bin/helm2
rm -rf linux-amd64

tar -xzf /tmp/helm.tar.gz
sudo mv linux-amd64/helm /usr/local/bin/helm
rm -rf linux-amd64

curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl


