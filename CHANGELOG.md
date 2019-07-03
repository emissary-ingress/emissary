# Datawire build-aux CHANGELOG

 - 2019-07-03: `go-mod.mk`: `.opensource.tar.gz` files are still part
   of `make build`, but no longer part of `make go-build`.

 - 2019-07-03: Rewrite the `go-opensource` Bash script as
   `go-mkopensource` in Go.

 - 2019-07-03: Migrate from `curl` to `go.mod`.
 - 2019-07-03: BREAKING CHANGE: Move executables to be in
   `./build-aux/bin/` instead of directly in `./build-aux/`.  Each of
   these programs now has a variable to refer to it by, instead of
   having to hard-code the path.  It is also no longer valid to use
   one of those programs without depending on it first.
