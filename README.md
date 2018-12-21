# Datawire build-aux

This is a collection of Makefile snippets (and associated utilities)
for use in Datawire projects.

## How to use

Add `build-aux.git` as `build-aux/` in the git repository of the
project that you want to use this from.  I recommend that you do this
using `git subtree`, but `git submodule` is fine too.

Then, in your Makefile, write `include build-aux/FOO.mk` for each
common bit of functionality that you want to make use of.

### Using `git-subtree` to manage `./build-aux/`

 - Start using build-aux:

       $ git subtree add --squash --prefix=build-aux git@github.com:datawire/build-aux.git master

 - Update to latest build-aux:

       $ ./build-aux/build-aux-pull

 - Push "vendored" changes upstream to build-aux.git:

       $ ./build-aux/build-aux-push
