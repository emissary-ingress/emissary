#!/usr/bin/env bash
set -ex

# Load the images
docker image load < docker/images.tar
./docker/images.sh

stamp_docker () {
    tag="$1"
    stamp_base="$2"

    hash=$(docker image inspect --format='{{ .Id }}' "$tag")

    for stamp_file in ".$stamp_base.docker.stamp" "$stamp_base.docker" "$stamp_base.docker.tag.local"; do
        echo "Stamping $tag as docker/$stamp_file"
        echo "$hash" > "docker/$stamp_file"
        # sleep 1
    done

    tag_file="docker/$stamp_base.docker.tag.local"
    echo "Adding tag to $tag_file"
    echo "$tag" >> "$tag_file"
}

stamp_image () {
    tag="$1"
    stamp_base="$2"

    hash=$(docker image inspect --format='{{ .Id }}' "$tag")

    for tarfile in "docker/$stamp_base.img.tar" "docker/.$stamp_base.img.tar.stamp"; do
	    echo "Copying $tag to $tarfile"
	    docker save "$tag" > "$tarfile"
	done
}

# ORDER MATTERS HERE
stamp_image frolvlad/alpine-glibc:alpine-3.15_glibc-2.34 base 	# This MUST be frolvlad, not emissary.local/base
stamp_image emissary.local/kat-client    kat-client
stamp_image emissary.local/kat-server    kat-server

stamp_docker emissary.local/base-envoy   base-envoy
stamp_docker emissary.local/base-python  base-python
stamp_docker emissary.local/base-pip     base-pip
stamp_docker emissary.local/base         base
stamp_docker emissary.local/emissary     emissary
stamp_docker emissary.local/kat-client   kat-client
stamp_docker emissary.local/kat-server   kat-server

# # Resume the build container
# if [[ -z "$DEV_REGISTRY" ]]; then
#     export DEV_REGISTRY=127.0.0.1:31000
#     export BASE_REGISTRY=docker.io/datawiredev
# fi
# rm -f docker/container.txt docker/container.txt.stamp
# make docker/container.txt
# docker run \
#   --rm \
#   --volume=/var/run/docker.sock:/var/run/docker.sock \
#   --user=root \
#   --entrypoint=rsync $(cat docker/snapshot.docker) \
#     -a -xx --exclude=/etc/{resolv.conf,hostname,hosts} --delete \
#     --blocking-io -e 'docker exec -i --user=root' / "$(cat docker/container.txt):/"
# docker exec "$(cat docker/container.txt)" rm -f /buildroot/image.dirty
# # Load the cache volume
# docker run \
#   --rm \
#   --volumes-from=$(cat docker/container.txt) \
#   --volume="$PWD/docker":/mnt \
#   --user=root \
#   --workdir=/home/dw \
#   --entrypoint=tar $(cat docker/snapshot.docker) -xf /mnt/volume.tar
# rm -f docker/volume.tar
