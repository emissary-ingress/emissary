(function () {
    "use strict";
    return {
        // Production configuration
        "backends": ["./backends/repeater"],
        "flushInterval": 10000,

        // Development configuration
        // "debug": true,
        // "backends": ["./backends/repeater", "./backends/console"],
        // "flushInterval": 1000,

        "repeater": [ { "host": "statsd-sink", "port": 8125 } ],
        "repeaterProtocol": "udp4"
    };
})();
