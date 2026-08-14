#!/usr/bin/env bash
#
# Run a kat-client query set until every jq assertion passes.
#
#   probe.sh <queries.json> <jq-expr>
#   probe.sh <queries.json> <label> <jq-expr> [<label> <jq-expr> ...]
#
# queries.json is a normal kat-client input file. Relative URLs ("/foo/") are
# prefixed with $PROBE_BASE (default $GATEWAY_URL); absolute ones are left
# alone, so TLS/alternate-port fixtures can spell out their own scheme+host.
#
# Each <jq-expr> is evaluated against the kat-client output array. This is the
# replacement for KAT's Python `check()`: `expected=404` becomes
# `.[0].result.status == 404`, and `r.backend.name == ...` becomes
# `.[0].result.json.backend == "..."`.
#
# Prefer the labelled form once a fixture has more than a couple of assertions.
# A single expression ANDing 69 clauses together -- which is what porting
# t_extauth.py faithfully would mean -- returns one bit on failure and tells you
# nothing about which clause broke. Labelled assertions are evaluated
# individually and the failing ones are named.
#
# Note the header casing asymmetry when writing expressions. kat-server
# lowercases the *request* headers it echoes back but leaves *response* headers
# canonical, so:
#
#   .[0].result.json.request.headers["x-bar"]   # request  -> lowercase
#   .[0].result.headers["X-Bar"]                # response -> canonical
#
# ponytail: retry loop lives here, not in each fixture. Envoy config
# propagation is async and chainsaw does not retry `script` steps.
set -uo pipefail

usage() {
    echo "usage: probe.sh <queries.json> <jq-expr>" >&2
    echo "       probe.sh <queries.json> <label> <jq-expr> [<label> <jq-expr> ...]" >&2
    exit 2
}

queries="${1:-}"
[[ -n "$queries" ]] || usage
shift

labels=() exprs=()
if [[ $# -eq 1 ]]; then
    labels=("assertion"); exprs=("$1")
elif [[ $# -ge 2 && $(($# % 2)) -eq 0 ]]; then
    while [[ $# -gt 0 ]]; do
        labels+=("$1"); exprs+=("$2"); shift 2
    done
else
    usage
fi

base="${PROBE_BASE:-${GATEWAY_URL:?GATEWAY_URL or PROBE_BASE must be set}}"
kat="${KAT_CLIENT:?KAT_CLIENT must be set}"
attempts="${PROBE_ATTEMPTS:-30}"
interval="${PROBE_INTERVAL:-2}"
total=${#exprs[@]}

resolved=$(jq --arg base "$base" \
    '[.[] | .url |= (if startswith("http") then . else $base + . end)]' \
    "$queries") || exit 1

summary='[.[] | {url: .url, status: .result.status, backend: .result.json.backend, error: .result.error}]'

out=""
failed=()
for i in $(seq 1 "$attempts"); do
    out=$(printf '%s' "$resolved" | "$kat" 2>/tmp/probe-stderr)

    failed=()
    for n in "${!exprs[@]}"; do
        jq -e "${exprs[n]}" <<<"$out" >/dev/null 2>&1 || failed+=("$n")
    done

    if [[ ${#failed[@]} -eq 0 ]]; then
        echo "ok (attempt ${i}): ${total}/${total} assertions passed"
        jq -c "$summary" <<<"$out" 2>/dev/null || true
        exit 0
    fi

    echo "attempt ${i}: $((total - ${#failed[@]}))/${total} passing" \
         "(first failure: ${labels[${failed[0]}]})"
    sleep "$interval"
done

{
    echo "probe failed after ${attempts} attempts:" \
         "${#failed[@]}/${total} assertions still failing"
    for n in "${failed[@]}"; do
        echo "  FAIL  ${labels[n]}"
        # Collapse the expression onto one line; they are often written across
        # several in YAML and are easier to match against the fixture this way.
        echo "        $(tr -s ' \n' ' ' <<<"${exprs[n]}" | sed 's/^ //;s/ $//')"
    done
    echo "response summary:"
    jq "$summary" <<<"$out" 2>/dev/null || true
    echo "full response:"
    jq . <<<"$out" 2>/dev/null || echo "$out"
    echo "--- kat-client stderr ---"
    cat /tmp/probe-stderr 2>/dev/null || true
} >&2
exit 1
