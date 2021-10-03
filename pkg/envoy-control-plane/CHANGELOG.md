# Changelog

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
