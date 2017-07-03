# Changelog

## [Unreleased]
[Unreleased]: https://github.com/datawire/ambassador/compare/v0.10.0...

### Added
- Changelog


## [0.10.0] June 30, 2017
[0.10.0]: https://github.com/datawire/ambassador/compare/v0.9.1...v0.10.0


## [0.9.1] June 28, 2017
[0.9.1]: https://github.com/datawire/ambassador/compare/v0.9.0...v0.9.1


## [0.9.0] June 23, 2017
[0.9.0]: https://github.com/datawire/ambassador/compare/v0.8.12...v0.9.0


## [0.8.12] June 07, 2017
[0.8.12]: https://github.com/datawire/ambassador/compare/v0.8.11...v0.8.12


## [0.8.11] May 24, 2017
[0.8.11]: https://github.com/datawire/ambassador/compare/v0.8.10...v0.8.11


## [0.8.10] May 24, 2017
[0.8.10]: https://github.com/datawire/ambassador/compare/v0.8.9...v0.8.10


## [0.8.9] May 23, 2017
[0.8.9]: https://github.com/datawire/ambassador/compare/v0.8.8...v0.8.9


## [0.8.8] May 23, 2017
[0.8.8]: https://github.com/datawire/ambassador/compare/v0.8.7...v0.8.8


## [0.8.7] May 23, 2017
[0.8.7]: https://github.com/datawire/ambassador/compare/v0.8.6...v0.8.7


## [0.8.6] May 05, 2017
[0.8.6]: https://github.com/datawire/ambassador/compare/v0.8.5...v0.8.6


## [0.8.5] May 05, 2017
[0.8.5]: https://github.com/datawire/ambassador/compare/v0.8.4...v0.8.5


## [0.8.4] May 05, 2017
[0.8.4]: https://github.com/datawire/ambassador/compare/v0.8.3...v0.8.4


## [0.8.3] May 05, 2017
[0.8.3]: https://github.com/datawire/ambassador/compare/v0.8.2...v0.8.3


## [0.8.2] May 04, 2017
[0.8.2]: https://github.com/datawire/ambassador/compare/v0.8.1...v0.8.2


## [0.8.1] May 04, 2017
[0.8.1]: https://github.com/datawire/ambassador/compare/v0.8.0...v0.8.1


## [0.8.0] May 02, 2017
[0.8.0]: https://github.com/datawire/ambassador/compare/v0.7.0...v0.8.0


## [0.7.0] May 01, 2017
[0.7.0]: https://github.com/datawire/ambassador/compare/v0.6.0...v0.7.0

### Added
- Ambassador can rewrite the request URL path prefix before forwarding the request to your service
- Ambassador supports additional stats aggregators: Datadog, Grafana

### Changed
- _Services_ are now known as _mappings_
- Minikube is supported again


## [0.6.0] April 28, 2017
[0.6.0]: https://github.com/datawire/ambassador/compare/v0.5.2...v0.6.0

### Removed
- The Ambassador SDS has been removed; Ambassador routes to service names


## [0.5.2] April 26, 2017
[0.5.2]: https://github.com/datawire/ambassador/compare/v0.5.0...v0.5.2

### Added
- Ambassador includes a local `statsd` so that full stats from Envoy can be collected and pushed to a stats aggregator (Prometheus is supported)

### Changed
- It's easier to develop Ambassador thanks to improved build documentation and `Makefile` fixes


## [0.5.0] April 13, 2017
[0.5.0]: https://github.com/datawire/ambassador/compare/v0.4.0...v0.5.0

### Added
- Ambassador supports inbound TLS
- YAML for a demo user service is now included

### Changed
- The `geturl` script supports Minikube and handles AWS better
- Documentation and code cleanup


## [0.4.0] April 07, 2017
[0.4.0]: https://github.com/datawire/ambassador/compare/v0.3.3...v0.4.0

### Changed
- Ambassador now reconfigures Envoy automatically once changes have settled for five seconds


## [0.3.3] April 07, 2017
[0.3.3]: https://github.com/datawire/ambassador/compare/v0.3.2...v0.3.3

### Changed
- Mappings no longer require specifying the port as it is not needed


## [0.3.2] April 07, 2017
[0.3.2]: https://github.com/datawire/ambassador/compare/v0.3.1...v0.3.2

### Changed
- Envoy stats and Ambassador stats are separate

### Fixed
- SDS does the right thing with unnamed ports


## [0.3.1] April 06, 2017
[0.3.1]: https://github.com/datawire/ambassador/compare/v0.3.0...v0.3.1

### Changed
- Split up `ambassador.py` into multiple files
- Switch to a debug build of Envoy


## [0.3.0] April 06, 2017
[0.3.0]: https://github.com/datawire/ambassador/compare/v0.2.0...v0.3.0

### Added
- Basic interpretation of cluster stats


## [0.2.0] April 06, 2017
[0.2.0]: https://github.com/datawire/ambassador/compare/v0.1.9...v0.2.0

### Added
- Envoy stats accessible through Ambassador


## [0.1.9] April 03, 2017
[0.1.9]: https://github.com/datawire/ambassador/compare/v0.1.8...v0.1.9

### Fixed
- Ambassador keeps running with an empty services list (part 2)


## [0.1.8] April 03, 2017
[0.1.8]: https://github.com/datawire/ambassador/compare/v0.1.7...v0.1.8

### Changed
- Conditionalize Docker push


## [0.1.7] April 03, 2017
[0.1.7]: https://github.com/datawire/ambassador/compare/v0.1.6...v0.1.7

### Changed
- Updated to current Envoy and pinned the Envoy version
- Use Bumpversion for version management


## [0.1.6] April 03, 2017
[0.1.6]: https://github.com/datawire/ambassador/compare/v0.1.5...v0.1.6

### Changed
- Ambassador configuration on `/ambassador-config/` prefix rather than exposed on port 8001


## [0.1.5] March 31, 2017
[0.1.5]: https://github.com/datawire/ambassador/compare/v0.1.4...v0.1.5

### Fixed
- Ambassador SDS correctly handles ports


## [0.1.4] March 31, 2017
[0.1.4]: https://github.com/datawire/ambassador/compare/v0.1.3...v0.1.4

### Changed
- Ambassador keeps running with an empty services list
- Easier to run with [Telepresence]


## [0.1.3] March 31, 2017
[0.1.3]: https://github.com/datawire/ambassador/compare/82ed5e4...v0.1.3

### Added
- Initial Ambassador
- Ambassador service discovery service
- Documentation


Based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/). Ambassador follows [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

[Telepresence]: http://telepresence.io
