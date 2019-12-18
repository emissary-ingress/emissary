#!/bin/bash

printf "== Begin: Pushing Helm Chart =="

echo Packaging Helm Chart

helm package 

# Get name of package
export CHART_PACKAGE=$(ls *.tgz)

curl -o tmp.yaml -k -L https://getambassador.io/helm/index.yaml

helm repo index . --url https://getambassador.io/helm --merge tmp.yaml

echo Pushing chart to S3 bucket

aws s3api put-object \
  --bucket datawire-static-files \
  --key ambassador/$CHART_PACKAGE \
  --body $CHART_PACKAGE

aws s3api put-object \
  --bucket datawire-static-files \
  --key ambassador/index.yaml \
  --body index.yaml

echo Cleanup

rm tmp.yaml index.yaml $CHART_PACKAGE

printf "== End: Pushing Helm Chart =="

