# Hacking on build-aux.git

## Misc notes

 - Any `.go` files should say `// +build ignore` to prevent `go list
   ./...` from picking them up.
 - If you have a dependency on another `.mk` file includes, include it
   with `include $(dir $(lastword $(MAKEFILE_LIST)))/common.mk`.
 - `.PHONY` targets that you wish to be user-visible should have a `##
   Help text` usage comment.  See `help.mk` for more information.

## Naming conventions

 - `check` and `check-FOO` are `.PHONY` targets that run tests.
 - `test-FOO` is an executable program that when run tests FOO.
   Perhaps the `check-FOO` Make target compiles and runs the
   `test-FOO` program.
 - `test` is the POSIX `test(1)` command.  Don't use it as a Makefile
   rule name.
 - (That is, use "check" as a *verb*, and "test" as a *noun*.)

## Compatibility

 - Everything should work with GNU Make 3.81 AND newer versions.
   * Avoid Make features introduced in 3.82 or later.
   * The 3.81→3.82 changed precedence of pattern rules from
     "parse-order-based" to "stem-length-based".  Be careful that your
     pattern rules work in BOTH systems.
 - Requires `go` 1.11 or newer.
 - Using `--` to separate positional arguments isn't POSIX, but is
   implemented in `getopt(3)` and `getopt_long(3)` in every major libc
   (including macOS).  Therefore, `--` working is a reasonable base
   assumption.  Known exceptions:
    * macOS `chmod`
