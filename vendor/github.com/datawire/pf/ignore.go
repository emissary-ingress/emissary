package pf

// `go mod vendor` will prune unused directories; because there are no
// .go files in the libkern/ or net/ directories, and because even if
// there were no other .go file import them, `go mod vendor` won't
// include them, which is wrong, because we need the .h files in them.
// So have some "empty" ignore.go files to trick `go mod vendor` in to
// including the .h files.
import (
	_ "github.com/datawire/pf/libkern"
	_ "github.com/datawire/pf/net"
)
