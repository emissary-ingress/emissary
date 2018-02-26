(function () {
    "use strict";
    return {
        // Here's some documentation for this:
        // https://github.com/etsy/statsd/blob/master/exampleConfig.js

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
