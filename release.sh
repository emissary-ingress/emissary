#!/bin/bash

# This script downloads the release image and pushes the release image to the
# actual Pro repository.

docker pull quay.io/datawire/ambassador-ratelimit:0.0.1
docker tag quay.io/datawire/ambassador-ratelimit:0.0.1 quay.io/datawire/ambassador_pro:ratelimit-0.1
docker push quay.io/datawire/ambassador_pro:ratelimit-0.1