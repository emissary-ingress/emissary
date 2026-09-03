#!/usr/bin/env bash
#
# Slot management for the e2e suite.
#
#   slots.sh list
#   slots.sh install <chart.tgz> <image-repo> <image-tag>
#   slots.sh uninstall
#   slots.sh check <fixtures-dir>
#   slots.sh run <chainsaw-bin> <config> <fixtures-dir>
#
# A slot is an isolated Emissary install, scoped by AMBASSADOR_ID. Called by
# both `make e2e/...` and test-images.yaml. See test/e2e/README.md.
set -euo pipefail

# <name>:<port-base>; a slot owns <base>+0 http, +1 https, +2 tcp. k3d fixes
# published ports at cluster-creation time, so the 8100-8159 block it reserves
# caps this at 15 slots.
SLOTS="${E2E_SLOTS:-slot1:8100 slot2:8104 slot3:8108 slot4:8112 slot5:8116 slot6:8120 slot7:8124 slot8:8128}"

NAMESPACE="${E2E_NAMESPACE:-emissary}"
GATEWAY_URL="${E2E_GATEWAY_URL:-http://localhost}"
DEFAULT_PARALLEL="${E2E_DEFAULT_PARALLEL:-8}"

cmd_list() {
    for f in $SLOTS; do
        echo "${f%%:*} ${f##*:}"
    done
}

cmd_install() {
    local chart="${1:?usage: slots.sh install <chart.tgz> <image-repo> <image-tag>}"
    local repo="${2:?missing image repo}"
    local tag="${3:?missing image tag}"

    # module.enabled=false and a per-slot ingressClassResource.name both avoid
    # colliding with the base release; see README.md. Installs run concurrently,
    # with output buffered per slot.
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

    for f in $SLOTS; do
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
    for f in $SLOTS; do
        local name="${f%%:*}"
        helm uninstall "emissary-${name}" --namespace "${NAMESPACE}-${name}" || true
        kubectl delete namespace "${NAMESPACE}-${name}" --ignore-not-found
    done
}

# Fixtures declare whether they need isolation, not which slot; see README.md.
VALID_KINDS=" shared exclusive "

# Two reads rather than one packed line: `read` collapses an empty leading
# field, which is exactly the missing-label case the guard below reports.
slot_kind() { sed -n 's/^ *slot: *//p' "$1" | head -n1; }
slot_name() { sed -n 's/^ *name: *//p' "$1" | head -n1; }

# A fixture with a bad label matches no shard and is skipped in silence, so
# reject it loudly instead.
cmd_check() {
    local fixtures="${1:?usage: slots.sh check <fixtures-dir>}"
    local bad=0 t kind name shared=0 exclusive=0

    for t in "$fixtures"/*/chainsaw-test.yaml; do
        [[ -e "$t" ]] || continue
        kind="$(slot_kind "$t")"; name="$(slot_name "$t")"
        if [[ -z "$kind" ]]; then
            echo "error: $t has no 'slot' label; add 'shared' or 'exclusive' under metadata.labels" >&2
            bad=1
        elif [[ "$VALID_KINDS" != *" ${kind} "* ]]; then
            echo "error: $t has unknown slot kind '${kind}'; expected one of:${VALID_KINDS}" >&2
            bad=1
        elif [[ -z "$name" ]]; then
            echo "error: $t has no metadata.name; the runner needs it to target the fixture" >&2
            bad=1
        elif [[ ! "$name" =~ ^[a-z0-9][a-z0-9-]*$ ]]; then
            # Names go into a generated regex; reject metacharacters rather
            # than escape them.
            echo "error: $t name '${name}' must match ^[a-z0-9][a-z0-9-]*\$" >&2
            bad=1
        else
            [[ "$kind" == shared ]] && shared=$((shared + 1)) || exclusive=$((exclusive + 1))
        fi
    done

    if [[ "$bad" -ne 0 ]]; then
        echo "error: fix the labels above, or those fixtures never run" >&2
        return 1
    fi
    echo "+ slot labels ok (${shared} shared, ${exclusive} exclusive)"
}

# One chainsaw process per shard, all concurrent: `shared` runs every shared
# fixture at --parallel N on the base Emissary, and each slot shard runs its
# dealt exclusive fixtures at --parallel 1. Shards select by name via
# --include-test-regex, which chainsaw feeds to Go's -run as `chainsaw/<name>`.
# Each exports SLOT for the fixture's ambassador_id. See README.md.
cmd_run() {
    local chainsaw="${1:?usage: slots.sh run <chainsaw-bin> <config> <fixtures-dir>}"
    local config="${2:?missing config}"
    local fixtures="${3:?missing fixtures dir}"

    cmd_check "$fixtures"

    # Deal the exclusive fixtures across the slots.
    local slot_names=() slot_bases=() f
    for f in $SLOTS; do slot_names+=("${f%%:*}"); slot_bases+=("${f##*:}"); done
    local nslots=${#slot_names[@]}
    (( nslots > 0 )) || { echo "error: no slots configured" >&2; return 1; }

    local assigned=() i=0 t kind name
    for ((i = 0; i < nslots; i++)); do assigned[i]=""; done
    i=0
    for t in "$fixtures"/*/chainsaw-test.yaml; do
        [[ -e "$t" ]] || continue
        kind="$(slot_kind "$t")"; name="$(slot_name "$t")"
        [[ "$kind" == exclusive ]] || continue
        local idx=$((i % nslots))
        assigned[idx]="${assigned[idx]}|${name}"
        i=$((i + 1))
    done
    echo "+ dealt ${i} exclusive fixture(s) across ${nslots} slot(s)"

    local logdir shards=() pids=()
    logdir="$(mktemp -d)"

    run_shard() {
        local shard="$1" slot="$2" url="$3" tcp_url="$4" parallel="$5"
        shift 5
        set +e
        SLOT="$slot" GATEWAY_URL="$url" GATEWAY_TCP_URL="$tcp_url" \
            "$chainsaw" test \
                --config "$config" \
                --parallel "$parallel" \
                "$@" \
                "$fixtures" >"${logdir}/${shard}.log" 2>&1
        echo "$?" >"${logdir}/${shard}.rc"
        set -e
    }

    run_shard shared default "$GATEWAY_URL" "${GATEWAY_URL}:6789" "$DEFAULT_PARALLEL" \
        --selector "slot=shared" &
    pids+=("$!"); shards+=(shared)

    for ((i = 0; i < nslots; i++)); do
        # A slot with nothing dealt to it gets no process at all; passing an
        # empty regex would select every test rather than none.
        [[ -n "${assigned[i]}" ]] || continue
        local name="${slot_names[i]}" base="${slot_bases[i]}"
        run_shard "$name" "$name" "${GATEWAY_URL}:${base}" "${GATEWAY_URL}:$((base + 2))" 1 \
            --include-test-regex "chainsaw/(${assigned[i]#|})\$" &
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
        echo "usage: slots.sh {list|install|uninstall|check|run} ..." >&2
        exit 2
        ;;
esac
