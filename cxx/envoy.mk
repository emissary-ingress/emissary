# This file is a stub until I port over the real Envoy build routine.

BASE_IMAGE.envoy = quay.io/datawire/ambassador-base:envoy-6.6e6ae35f214b040f76666d86b30a6ad3ceb67046.dbg

base-envoy.docker.stamp: preflight
	docker run --rm --entrypoint=true $(BASE_IMAGE.envoy)
	docker image inspect $(BASE_IMAGE.envoy) --format='{{ .Id }}' > $@
