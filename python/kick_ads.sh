#!/bin/bash
# Usage: [DIAGD_ONLY=y] ENVOY_DIR=<envoy_dir> kick_ads.sh <ambex-pid> <envoy-flags...>

if [[ -n "$DIAGD_ONLY" ]]; then
    echo "Not starting, since in diagd-only mode."
    exit 0
fi

arg_ambex_pid="$1"
arg_envoy_flags=("${@:2}")

envoy_pid_file="${ENVOY_DIR}/envoy.pid"

if [[ ! -r "$envoy_pid_file" ]] || ! kill -0 $(cat "${envoy_pid_file}"); then
    # Envoy isn't running. Start it.
    envoy "${arg_envoy_flags[@]}" &
    envoy_pid="$!"
    echo "KICK: started Envoy as PID $envoy_pid"
    echo "$envoy_pid" > "$envoy_pid_file"
fi

# Once envoy is running, poke Ambex.
echo "KICK: kicking ambex"
kill -HUP "$arg_ambex_pid"
