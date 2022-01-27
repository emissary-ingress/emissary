# go-mkopensource

`go-mkopensource` is a program for generating reports of the libraries
used by a piece of Go code, in order to be in compliance with the
attribution requirements of various opensource licenses.

## Building

You may use `go get github.com/datawire/go-mkopensource`, clone the
repo and run `go build .`, or any of the other usual ways of building
a Go program; there is nothing special about `go-mkopensource`.

## Running

TL;DR: run one of

```shell
go-mkopensource --gotar=/path/to/go1.17.2.src.tar.gz --package=mod --output-format=txt >OPENSOURCE.md
go-mkopensource --gotar=/path/to/go1.17.2.src.tar.gz --package=mod --output-format=tar --output-name=mything >mything.OPENSOURCE.tar.gz
#               \________________  ________________/ \_____  ____/ \_________________________________  _______________________________/
#                                \/                        \/                                        \/
#                         Getting set up           Target to describe                          Output format
```

Let's now look at those flags piece-by-piece.

### Getting set up

There is one piece of knowledge that `go-mkopensource` cannot detect:
Which version of the Go standard library it should be referencing.
Before running `go-mkopensource`, you will need to download the source
tarball for the relevent version of Go.  For example for Go 1.17.2,
you need to download https://dl.google.com/go/go1.17.2.src.tar.gz .
When you run `go-mkopensource`, you will need to tell it the download
location by passing it the `--gotar` flag to point it at the download
path.  For example, `--gotar=$HOME/Downloads/go1.17.2.tar.gz`.

### Target to describe

The `--package=` flag tells `go-mkopensource` which Go packages it
should produce a report for.  You can either

 - pass it an string that you would pass to `go list` like `./...` or
   `./cmd/mything`, or
 - pass the special value `mod` to describe the entire Go module of
   the current directory.

When passing a `go list` string, the behavior matches `go list`: What
gets returned may depend on `GOOS` and `GOARCH`.  You may set those
environment variables to affect what gets returned.  This means that
by default the report would not include dependencies that are only
needed on a platform other than the current one.

On the other hand, `--package=mod` considers *all* operating systems
and architectures, including dependencies in the report even if they
are only needed on a single platform.

### Output format

There are two modes of operation:

 1. `--output-format=txt` which produces a short-ish textual report of
    all of the dependencies, their versions, and their licenses.
 2. `--output-format=tar` which produces a gzipped-tarball containing
    both the `OPENSOURCE.md` file that `--output-format=txt` generates
    (plus a footer reading "The appropriate license notices and source
    code are in correspondingly named directories."), and a directory
    for each dependency, containing any nescessary license notices and
    source code.

Many licenses require the author to be credited, the full license text
to be included, a notice to be included, or even (in the case of the
MPL) a subset of the source code to be included.  This is what
`--output-format=tar` is for; the `--output-format=txt` output alone
is usually not sufficient to be in compliance with the licenses.

In both modes, it writes the output to stdout; it is up to you to
redirect this output to a file.

#### `--output-format=tar`

In the Windows world, it is normal for a `.zip` file to include files
directly inside of it in the root.  However, in the Unix world, it is
considered rude to have files directly inside of a `.tar` file; they
should all be inside of a directory in the tarball, the name of that
directory reflecting the expected name of the `.tar` file.

This is what the `--output-name=` flag is for, it tells
`go-mkopensource` what to use for the name of the directory inside of
the tarball (since it does not know the name of the file that you are
directing the output to).

## Using as a library

The [`github.com/datawire/go-mkopensource/pkg/detectlicense`][detectlicense]
package is good at detecting the licenses in a file, and may be reused
(for example, by [Emissary's `py-mkopensource`][py-mkopensource]).

[detectlicense]: https://pkg.go.dev/github.com/datawire/go-mkopensource/pkg/detectlicense
[py-mkopensource]: https://github.com/emissary-ingress/emissary/blob/master/tools/src/py-mkopensource/main.go

## Design

There are many existing packages to do license detection, such as
[go-license-detector][] or GitHub's [licensee][].  The reason these
are not used is that they are meant to be _informative_, they provide
"best effort" identification of the license.

`go-mkopensource` isn't meant to just be _informative_, it is meant to
be used for _compliance_, if it has any reason at all to be even a
little skeptical of a result, rather than returnit its best guess, it
blows up in your face, asking a human to verify the result.

[go-license-detector]: https://github.com/go-enry/go-license-detector
[licensee]: https://github.com/licensee/licensee
