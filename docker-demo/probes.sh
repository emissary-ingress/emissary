#!/bin/sh

mark_alive() {
    last_alive=$(date +%s)

    if [ -n "$log_alive" ]; then
        echo "PROBES: alive at $last_alive"
        log_alive=
    fi
}

mark_ready() {
    last_ready=$(date +%s)   

    if [ -n "$log_ready" ]; then
        echo "PROBES: ready at $last_ready"
        log_ready=
    fi
}

time_since_alive() {
    now=$(date +%s)
    echo $(($now - $last_alive))
}

time_since_ready() {
    now=$(date +%s)
    echo $(($now - $last_alive))
}

alive() {
    curl -s -o /dev/null -f http://localhost:8877/ambassador/v0/check_alive
}

ready() {
    curl -s -o /dev/null -f http://localhost:8877/ambassador/v0/check_ready
}

# Initialize...
mark_alive
mark_ready
log_alive=YES
log_ready=YES

time_to_die=

while [ -z "$time_to_die" ]; do
    sleep 3

    if alive; then mark_alive; fi
    if ready; then mark_ready; fi

    tsa=$(time_since_alive)
    tsr=$(time_since_ready)

    if [ $tsa -ge 60 ]; then
        echo "PROBES: time since alive $tsa, exiting"
        time_to_die=YES
    elif [ $tsa -ge 10 ]; then
        echo "PROBES: time since alive $tsa, flagging"
        log_alive=YES        
    fi

    if [ $tsr -ge 60 ]; then
        echo "PROBES: time since ready $tsr, exiting"
        time_to_die=YES
    elif [ $tsr -ge 10 ]; then
        echo "PROBES: time since ready $tsr, flagging"
        log_alive=YES
    fi
done

exit 1
