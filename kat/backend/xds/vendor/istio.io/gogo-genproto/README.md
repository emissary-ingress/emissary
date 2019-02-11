# gogo-genproto

These are gogoslick compiled versions of widely-used protos.

## Generate Files

To regenerate, issue a `make` command. You will need to have the following installed:
- `curl` (fetching source protos)
- `protoc` (proto-compiler) with a version >=3.5.0
    - [Releases](https://github.com/google/protobuf/releases/)
- [`dep`](https://github.com/golang/dep) (managing gogo binaries)
    - [Releases](https://github.com/golang/dep/releases)
