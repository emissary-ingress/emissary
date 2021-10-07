include $(dir $(lastword $(MAKEFILE_LIST)))tools.mk

K3S_VERSION      = 1.21.5+k3s1
K3D_CLUSTER_NAME =
K3D_ARGS         = --k3s-server-arg=--no-deploy=traefik --k3s-server-arg=--kubelet-arg=max-pods=255
# This is modeled after
# https://github.com/nolar/setup-k3d-k3s/blob/v1.0.7/action.sh#L70-L77 and
# https://github.com/nolar/setup-k3d-k3s/blob/v1.0.7/action.yaml#L34-L46
ci/setup-k3d: $(tools/k3d) $(tools/kubectl)
	$(tools/k3d) cluster create --wait --image=docker.io/rancher/k3s:v$(subst +,-,$(K3S_VERSION)) $(K3D_ARGS)
	while ! $(tools/kubectl) get serviceaccount default >/dev/null; do sleep 1; done
	$(tools/kubectl) version
.PHONY: ci/setup-k3d
