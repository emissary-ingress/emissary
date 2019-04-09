# Load Testing APro

The `./bin_linux_amd64/max-load` program is the basis of my
load-testing efforts.  It is built on top of the library form of
[vegeta][].  It will attempt to determine latency as a function of
RPS, and determine the maximum RPS that the service can support.  The
`./bin_linux_amd64/max-load --help` text should be helpful.

[vegeta]: https://github.com/tsenart/vegeta

The `./test.sh` script calls `max-load` with a variety of parameters
to test a buncha situations.
