# Changelog

## Release v0.10.3

### Changed

- Documentation rewrite in the `/docs` folder (#520)
- Updated go module version to 1.16 (#536)
- Exposed `ApiTypePrefix` (#553)
- Envoy Protos Commit SHA: `97dfffde06115e58261dbab3707ad70d5c86ba1f`

### Added

- Initial support of the Virtual Host Discovery Service VHDS (#529)
- Added linear cache method `UpdateResources` (#546)

### Fixed

- Scoped routes consistency check (#515)
- Scoped routes references (#518)
- Fixed go routine leaks in server unit tests (#519)
- Linear cache no longer requires linear time for applying delta updates (#547)

## Release v0.10.1

### Changed

- Envoy Protos Commit SHA: `9cc74781d818aaa58b9cca9602fe8dc62181…`
### Fixed

- Release to fix broken `GOSUMDB` checksum when using `v0.10.0`. Please pin to this release and ignore `v0.10.0`.


## Release v0.10.0

### Added

- Added snapshot support in the Linear cache (#437) 
- Added CI linting support (#455)
- Incremental xDS support for Linear and Mux caches (#459)
- Added Extension Configs support (#417)
- Added a default cache logger (#483)
- Added Scoped Routes Discovery Service - SRDS (#495)

### Changed

- Removed linearization in server API to preserve cache ordering (#443)
- SetSnapshot now takes a `context` (#474)
- Delta xDS now responds immediately for the first wildcard request in a delta stream if the corresponding snapshot exists and the response is empty (#473)
- Reworked snapshot API to faciliate additional xDS resources without changes (#484)
- Delta xDS won't delete non-existent resources in wildcard mode (#488)
- Simple cache now holds a read lock when cancelling a snapshot watch (#507)

### Fixed

- Delta xDS not registering another watch after resource sent (#458)
- Fixed data race in Linear cache (#502)
- State of the World now tracks known resource names per caller stream (#508)


## Release v0.9.9

### Added

- Add snapshot support for ECDS (#379)
- Add cache support for xDS TTLs (#359)
- Add cache interfaces for incremental xDS (#408)
- Incremental simple cache implementation (#411)

### Changed

- Envoy APIs are at b6039234e526eeccdf332a7eb041729aaa1bc286
- Update dependencies to use `cncf/xds` instead of `cncf/udpa` (#404)
- Log ignoring a watch at warn level (#352)
- Removed support for V2 Envoy APIs in the server (#415)

### Fixed

- Go 1.16 compatibility fixes (#409)
- Fix a potential goroutine leak in stream handler (#430)

## Release v0.9.8

### Changed

- Envoy APIs are at 1d44c27ff7d4ebdfbfd9a6acbcecf9631b107e30
- server: exit receiver go routine when context is done
- cache: align struct fields

## Release v0.9.7

### Added

- secrets to the cache snapshots
- linearly versioned cache for a single type resources
- version prefix to the linear cache
- support for arbitrary type URLs in xDS server

### Changed

- Envoy APIs are at 241358e0ac7716fac24ae6c19c7dcea67357e70e
- split `server` package into `sotw` and `rest`

## Release v0.9.6

### Added

- introduce Passthrough resource type for a pre-serialized xDS response

### Changed

- Envoy APIs are at 73fc620a34135a16070083f3c94b93d074f6e59f
- update dependencies: protobuf to v1.4.2 and grpc to v1.27.0 to support protobuf v2 development
- protobufs are generated with protobuf v2 toolchain
- updates to the wellknown extension names to use non-deprecated versions
- use LoggersFuncs struct to reduce boilerplate in debug logging
- use CallbackFuncs struct to reduce boilerplate in server callbacks

## Release v0.9.5

### Added

- Added integration tests for v2 and v3 versions
- Cache implementation is replicated into xDS v2 and xDS v3 versions. You need to add to "v2" or "v3" suffix to imports to indicate which version to use (thanks @jyotimahapatra)

### Changed 

- Updated Envoy SHA to 34fcdef99633947543070d5eadf32867e940694e
- Module requirement downgraded to go1.11
- `ExtAuthz` well known filter names are updated to the new Envoy format

### Removed

- v3 cache implementation removed GetStatusInfo and GetStatusKeys functions from the interface

### Issues

- `set_node_on_first_message_only` may not work as expected due to an Envoy issue
