#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR="$CURR_DIR/.."

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

#########################################################################################

MANIFESTS_DIR="$CURR_DIR/tests/manifests/"

HELM2_INIT_YAML="$MANIFESTS_DIR/helm-init.yaml"

CHART_YAML="$TOP_DIR/Chart.yaml"

VALUES_DIR="$CURR_DIR/tests/values"
VALUES_TEMPLATES_DIR="$CURR_DIR/tests/values_templates"

VALUES_CI="$MANIFESTS_DIR/ci-default-values.yaml"

OSS_TAG="$(cat $CHART_YAML | grep ossVersion | sed s/'ossVersion: '/''/)"
export OSS_TAG

EXE_PROVIDER="$CURR_DIR/providers/$PROVIDER.sh"

command_exists "$EXE_KUBECTL" || EXE_KUBECTL="kubectl"
command_exists "$EXE_HELM2"   || EXE_HELM2="helm2"
command_exists "$EXE_HELM3"   || EXE_HELM3="helm3"

#########################################################################################

info "Performing some preflight checks..."
[ -d "$MANIFESTS_DIR"        ] || abort "no manifests dir found in $MANIFESTS_DIR"
[ -d "$VALUES_TEMPLATES_DIR" ] || abort "no values templates found in $VALUES_TEMPLATES_DIR"
[ -f "$HELM2_INIT_YAML"      ] || abort "no helm2 manifest found in $HELM2_INIT_YAML"
[ -n "$PROVIDER"             ] || abort "no kubernetes cluster provider specified in the PROVIDER env var"
[ -x "$EXE_PROVIDER"         ] || abort "no kubernetes provider script found in $EXE_PROVIDER"

command_exists "$EXE_KUBECTL" || abort "no kubectl executable found at $EXE_KUBECTL (use KUBETCL env var)"
command_exists "$EXE_HELM2"   || abort "no helm2 executable found at $EXE_HELM2 (use HELM2 env var)"
command_exists "$EXE_HELM3"   || abort "no helm3 executable found at $EXE_HELM3 (use HELM3 env var)"

DEV_KUBECONFIG="$($EXE_PROVIDER get-kubeconfig)"
KUBECONFIG=$DEV_KUBECONFIG
[ -n "$KUBECONFIG" ] || abort "no valid KUBECONFIG obtained from $PROVIDER"
export DEV_KUBECONFIG KUBECONFIG

$EXE_KUBECTL get svc 2>&1 > /dev/null || abort "cannot get services with KUBECONFIG=$KUBECONFIG"

#########################################################################################

info "Starting Helm tests..."

info "Generating values files"
rm -rf "$VALUES_DIR"
mkdir -p "$VALUES_DIR"
for file in $VALUES_TEMPLATES_DIR/* ; do
  replace_env_file $file "$VALUES_DIR/$(basename $file)"
done

info "Bootstrapping Helm installs"

info "Bootstrap Helm 2"
$EXE_KUBECTL apply -f $HELM2_INIT_YAML
$EXE_HELM2 version
$EXE_HELM2 init --service-account=tiller --wait

info "Bootstrap Ambassador release"
$EXE_HELM3 version
$EXE_HELM3 install ambassador "$TOP_DIR" --wait -f "$VALUES_CI" 2>&1 > /dev/null

for i in $MANIFESTS_DIR/tls.yaml $MANIFESTS_DIR/backend.yaml ; do
  $EXE_KUBECTL apply -f "$i"
done
wait_pod_running "-l app=quote" || abort "backend not available"

info "Testing Helm 3 releases"
for v_file in $VALUES_DIR/* ; do
  info "Upgrading the Ambassador release with new values file..."
  $EXE_HELM3 upgrade ambassador "$TOP_DIR" --wait -f "$v_file" 2>&1 > /dev/null
  passed "Release upgraded with $v_file"
  
  info "Testing we can reach the backend api"
  $EXE_KUBECTL port-forward service/ambassador 8443:443 2>&1 > /dev/null &
  wait_url https://localhost:8443/backend/ || abort "could not reach /backend (on 8443 for service/ambassador)"
  kill_background
done
passed "Testing Helm 3 releases"

info "Testing Helm 2 release"
$EXE_HELM2 install -n ambassador-helm2 "$TOP_DIR" -f "$MANIFESTS_DIR/helm2-values.yaml" --wait 2>&1 > /dev/null

info "Release installed with Helm 2"
$EXE_KUBECTL port-forward service/ambassador-helm2 9443:443 2>&1 > /dev/null &
wait_url https://localhost:9443/backend/ || abort "could not reach /backend (on 9443 for service/ambassador-helm2)"
passed "Testing Helm 2 releases"

info "Cleaning up..."

$EXE_KUBECTL delete -f $MANIFESTS_DIR/backend.yaml
kill_background

$EXE_HELM3 uninstall ambassador > /dev/null
$EXE_HELM2 del --purge "ambassador-helm2"

wait_pod_missing "-l app.kubernetes.io/instance=ambassador" || abort "pod still running"
passed "helm 3 chart uninstalled"

wait_pod_missing "-l app.kubernetes.io/instance=ambassador-helm2" || abort "pod still running"
passed "helm 2 chart uninstalled"

rm -rf "$VALUES_DIR"
