#!/usr/bin/env bash
#
# Run a kat-client query set until a jq assertion passes.
#
#   probe.sh <queries.json> <jq-expr>
#
# queries.json is a normal kat-client input file. Relative URLs ("/foo/") are
# prefixed with $PROBE_BASE (default $GATEWAY_URL); absolute ones are left
# alone, so TLS/alternate-port fixtures can spell out their own scheme+host.
#
# <jq-expr> is evaluated against the kat-client output array. This is the
# replacement for KAT's Python `check()`: `expected=404` becomes
# `.[0].result.status == 404`, and `r.backend.name == ...` becomes
# `.[0].result.json.backend == "..."`.
#
# ponytail: retry loop lives here, not in each fixture. Envoy config
# propagation is async and chainsaw does not retry `script` steps.
set -uo pipefail

queries="${1:?usage: probe.sh <queries.json> <jq-expr>}"
expr="${2:?usage: probe.sh <queries.json> <jq-expr>}"

base="${PROBE_BASE:-${GATEWAY_URL:?GATEWAY_URL or PROBE_BASE must be set}}"
kat="${KAT_CLIENT:?KAT_CLIENT must be set}"
attempts="${PROBE_ATTEMPTS:-30}"
interval="${PROBE_INTERVAL:-2}"

resolved=$(jq --arg base "$base" \
    '[.[] | .url |= (if startswith("http") then . else $base + . end)]' \
    "$queries") || exit 1

summary='[.[] | {url: .url, status: .result.status, backend: .result.json.backend, error: .result.error}]'

for i in $(seq 1 "$attempts"); do
    out=$(printf '%s' "$resolved" | "$kat" 2>/tmp/probe-stderr)
    if jq -e "$expr" <<<"$out" >/dev/null 2>&1; then
        echo "ok (attempt ${i}): ${expr}"
        jq -c "$summary" <<<"$out"
        exit 0
    fi
    echo "attempt ${i}: $(jq -c "$summary" <<<"$out" 2>/dev/null || head -c 300 <<<"$out")"
    sleep "$interval"
done

echo "probe failed after ${attempts} attempts" >&2
echo "assertion: ${expr}" >&2
echo "last response:" >&2
echo "$out" >&2
echo "--- kat-client stderr ---" >&2
cat /tmp/probe-stderr >&2 2>/dev/null || true
exit 1
