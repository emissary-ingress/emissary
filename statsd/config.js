(function () {
    "use strict";
    return {
        // Production configuration
        //"backends": ["./backends/repeater"],
        //"flushInterval": 10000,

        // Development configuration
        "debug": false,
        "backends": [ "./backends/repeater" ],
        // , "./backends/console"],
        "flushInterval": 10000,

        "repeater": [ { "host": "statsd-sink", "port": 8125 } ],
        "repeaterProtocol": "upd4"
    };
})();
