You need to run `go generate .` to generate `bindata.go` from
`bindata/` whenever you change any of the HTML/assets there.

For quick iteration, there's a stub HTTP server in `./main`.

I run

    go generate . && go run ./main

and hit "ctrl-c, up, enter" before I hit "refresh" in my browser.

This requires that you have `go-bindata` installed and in your path.
The `Makefile` has a rule for grabbing the appropriate version of
`go-bindata`.
