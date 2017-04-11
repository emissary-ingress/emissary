(function () {
    "use strict";
    return {
        "debug": true,
        "backends": ["./backends/repeater", "./backends/console"],
        "repeater": [ { "host": "statsd-sink", "port": 8125 } ],
        //"repeaterProtocol": "tcp"  // Default: "udp4"
        "repeaterProtocol": "upd4"
    };
})();
