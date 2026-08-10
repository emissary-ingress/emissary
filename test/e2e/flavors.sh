#!/usr/bin/env bash
#
# Flavor management for the e2e suite.
#
#   flavors.sh list
#   flavors.sh install <chart.tgz> <image-repo> <image-tag>
#   flavors.sh uninstall
#   flavors.sh check <fixtures-dir>
#   flavors.sh run <chainsaw-bin> <config> <fixtures-dir>
#
# A "flavor" is a pre-baked, isolated Emissary install. Most of the KAT suite
# can't share one Emissary because a Module / AuthService / TracingService
# configures the whole instance, so two such tests on one Emissary would
# clobber each other.
#
# Isolation is by AMBASSADOR_ID, not namespace: an Emissary only acts on CRDs
# whose `ambassador_id` matches its own, and a CRD that omits the field
# defaults to "default". So a flavor is just another helm release of the
# shipped chart with env.AMBASSADOR_ID set -- which also makes the chart stamp
# the matching ambassador_id onto its default Listeners.
#
# Both `make e2e/...` and .github/workflows/test-images.yaml call this, so the
# flavor list and its port math have exactly one definition.
set -euo pipefail

# <name>:<port-base>. A slot owns <base>+0 (http), +1 (https), +2 (tcp), with a
# 4-port stride. k3d fixes published ports at cluster-creation time and the
# cluster reserves 8100-8159, so 15 slots fit before the range has to widen
# *and* the cluster be recreated.
#
# Slots are deliberately anonymous. Nothing distinguishes one from another
# except its ports -- all the config (Module, AuthService, ...) comes from the
# fixture's own CRDs at runtime, so any pinned fixture could run on any slot.
# Naming them after what they test would encode a property they don't have.
# Genuine presets (an Emissary differing in container env, e.g. one with
# AMBASSADOR_SINGLE_NAMESPACE or LUA_SCRIPTS_ENABLED) do differ from each other
# and should get real names when they arrive: `single-namespace:8156`.
#
# Sizing: at ~28s per fixture, N slots clear the flavor-pinned backlog in
# (count/N * 28)s. Returns flatten past N=8, where the `default` shard becomes
# the constraint instead. The ceiling is runner CPU, not ports -- the chart
# requests 200m per Emissary and a standard 4-vCPU runner leaves room for
# roughly 16, so 8-12 is the practical band.
FLAVORS="${E2E_FLAVORS:-slot1:8100 slot2:8104 slot3:8108 slot4:8112 slot5:8116 slot6:8120 slot7:8124 slot8:8128}"

# `default` is the plain install on 80/443/6789 with no AMBASSADOR_ID. It picks
# up CRDs that set no ambassador_id, so fixtures needing no global config
# target it and can overlap freely -- hence a parallelism well above 1.
NAMESPACE="${E2E_NAMESPACE:-emissary}"
GATEWAY_URL="${E2E_GATEWAY_URL:-http://localhost}"
DEFAULT_PARALLEL="${E2E_DEFAULT_PARALLEL:-8}"

cmd_list() {
    for f in $FLAVORS; do
        echo "${f%%:*} ${f##*:}"
    done
}

cmd_install() {
    local chart="${1:?usage: flavors.sh install <chart.tgz> <image-repo> <image-tag>}"
    local repo="${2:?missing image repo}"
    local tag="${3:?missing image tag}"

    # Two overrides keep a flavor from colliding with the base release:
    #
    # module.enabled=false -- the chart otherwise ships a Module named
    # `ambassador` carrying this flavor's ambassador_id. Emissary keys its
    # module store by name alone, so that one would collide with the Module a
    # fixture creates: Config.safe_store renames the loser to
    # `ambassador.<namespace>` and get_module("ambassador") then returns
    # whichever won the race, silently dropping the fixture's config.
    #
    # ingressClassResource.name -- IngressClass is cluster-scoped and the chart
    # names it from a fixed value, so a second release dies with "cannot be
    # imported into the current release". It is the chart's only cluster-scoped
    # resource that isn't already release-scoped. The template stamps
    # getambassador.io/ambassador-id onto it, so a per-flavor name stays usable
    # by any future Ingress fixture.
    #
    # NOTE: `helm template` hides the IngressClass unless you pass
    # `--api-versions networking.k8s.io/v1/IngressClass`, because the template
    # is gated on .Capabilities. Verifying a flavor render without that flag
    # will not show this collision.
    # Installs run concurrently. They're independent helm releases into separate
    # namespaces, and each spends nearly all its ~35s blocked in `--wait`, so
    # serialising them made setup the most expensive step in the job (141s for
    # four, against 32s to actually run the fixtures) and made every extra slot
    # cost another 35s. Output is buffered per slot so the logs stay readable.
    local logdir pids=() names=()
    logdir="$(mktemp -d)"

    install_one() {
        local name="$1" base="$2"
        set +e
        helm upgrade --install "emissary-${name}" "$chart" \
            --namespace "${NAMESPACE}-${name}" --create-namespace \
            --set image.repository="$repo" \
            --set image.tag="$tag" \
            --set image.pullPolicy=IfNotPresent \
            --set replicaCount=1 \
            --set createDefaultListeners=true \
            --set env.AMBASSADOR_ID="$name" \
            --set module.enabled=false \
            --set ingressClassResource.name="ambassador-${name}" \
            --set "service.ports[0].name=http" \
            --set "service.ports[0].port=${base}" \
            --set "service.ports[0].targetPort=8080" \
            --set "service.ports[1].name=https" \
            --set "service.ports[1].port=$((base + 1))" \
            --set "service.ports[1].targetPort=8443" \
            --set "service.ports[2].name=tcp" \
            --set "service.ports[2].port=$((base + 2))" \
            --set "service.ports[2].targetPort=6789" \
            --set "service.ports[2].protocol=TCP" \
            --wait --timeout 5m \
            && kubectl -n "${NAMESPACE}-${name}" \
                rollout status "deploy/emissary-${name}" --timeout=2m
        echo "$?" >"${logdir}/${name}.rc"
        set -e
    }

    for f in $FLAVORS; do
        local name="${f%%:*}" base="${f##*:}"
        echo "+ installing slot '${name}' (http=${base} https=$((base + 1)) tcp=$((base + 2)))"
        install_one "$name" "$base" >"${logdir}/${name}.log" 2>&1 &
        pids+=("$!"); names+=("$name")
    done

    wait "${pids[@]}"

    local rc=0 src
    for name in "${names[@]}"; do
        src="$(cat "${logdir}/${name}.rc" 2>/dev/null || echo 1)"
        if [[ "$src" -ne 0 ]]; then
            echo "==================== slot ${name} FAILED (rc=${src}) ====================" >&2
            cat "${logdir}/${name}.log" >&2
            rc=1
        else
            echo "+ slot '${name}' ready"
        fi
    done

    rm -rf "$logdir"
    return "$rc"
}

cmd_uninstall() {
    for f in $FLAVORS; do
        local name="${f%%:*}"
        helm uninstall "emissary-${name}" --namespace "${NAMESPACE}-${name}" || true
        kubectl delete namespace "${NAMESPACE}-${name}" --ignore-not-found
    done
}

# Every fixture must carry a `flavor` label naming a shard that actually runs.
# Without this a typo means no shard's selector matches, and the fixture is
# skipped in silence -- a green suite that tested nothing.
cmd_check() {
    local fixtures="${1:?usage: flavors.sh check <fixtures-dir>}"
    local valid=" default"
    for f in $FLAVORS; do valid="${valid} ${f%%:*}"; done

    local bad=0 t flavor
    for t in "$fixtures"/*/chainsaw-test.yaml; do
        [[ -e "$t" ]] || continue
        flavor="$(sed -n 's/^ *flavor: *//p' "$t" | head -n1)"
        if [[ -z "$flavor" ]]; then
            echo "error: $t has no 'flavor' label (add one under metadata.labels)" >&2
            bad=1
        elif [[ " ${valid} " != *" ${flavor} "* ]]; then
            echo "error: $t has unknown flavor '${flavor}'; known:${valid}" >&2
            bad=1
        fi
    done
    if [[ "$bad" -ne 0 ]]; then
        echo "error: fix the labels above, or those fixtures never run" >&2
        return 1
    fi
    echo "+ flavor labels ok ($(find "$fixtures" -name chainsaw-test.yaml | wc -l | tr -d ' ') fixtures)"
}

# One chainsaw process per flavor plus one for `default`. A flavor's shard runs
# serially because its tests take turns owning that Emissary's global config;
# the shards themselves run concurrently. Output is buffered per shard so the
# logs don't interleave.
cmd_run() {
    local chainsaw="${1:?usage: flavors.sh run <chainsaw-bin> <config> <fixtures-dir>}"
    local config="${2:?missing config}"
    local fixtures="${3:?missing fixtures dir}"

    cmd_check "$fixtures"

    local logdir shards=() pids=()
    logdir="$(mktemp -d)"

    run_shard() {
        local name="$1" url="$2" tcp_url="$3" parallel="$4"
        set +e
        GATEWAY_URL="$url" GATEWAY_TCP_URL="$tcp_url" \
            "$chainsaw" test \
                --config "$config" \
                --selector "flavor=${name}" \
                --parallel "$parallel" \
                "$fixtures" >"${logdir}/${name}.log" 2>&1
        echo "$?" >"${logdir}/${name}.rc"
        set -e
    }

    run_shard default "$GATEWAY_URL" "${GATEWAY_URL}:6789" "$DEFAULT_PARALLEL" &
    pids+=("$!"); shards+=(default)

    for f in $FLAVORS; do
        local name="${f%%:*}" base="${f##*:}"
        run_shard "$name" "${GATEWAY_URL}:${base}" "${GATEWAY_URL}:$((base + 2))" 1 &
        pids+=("$!"); shards+=("$name")
    done

    wait "${pids[@]}"

    local rc=0 src
    for s in "${shards[@]}"; do
        echo "==================== shard: ${s} ===================="
        cat "${logdir}/${s}.log"
        src="$(cat "${logdir}/${s}.rc" 2>/dev/null || echo 1)"
        if [[ "$src" -ne 0 ]]; then
            echo "shard '${s}' FAILED (rc=${src})" >&2
            rc=1
        fi
    done

    rm -rf "$logdir"
    return "$rc"
}

case "${1:-}" in
    list)      shift; cmd_list "$@" ;;
    install)   shift; cmd_install "$@" ;;
    uninstall) shift; cmd_uninstall "$@" ;;
    check)     shift; cmd_check "$@" ;;
    run)       shift; cmd_run "$@" ;;
    *)
        echo "usage: flavors.sh {list|install|uninstall|check|run} ..." >&2
        exit 2
        ;;
esac
