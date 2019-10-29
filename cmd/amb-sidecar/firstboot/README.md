You need to run `go generate .` to generate `bindata.go` from
`bindata/` whenever you change any of the HTML/assets there.

This requires that you have `go-bindata` installed and in your path.

The `Makefile` has a rule for  grabbing the appropriate version of
`go-bindata` and calling `go generate .`:

    make pro-generate
