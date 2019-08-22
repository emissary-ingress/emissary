package main

import (
	"os"

	"github.com/spf13/cobra"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

var argparser = &cobra.Command{
	Use: os.Args[0],
}

func main() {
	argparser.Execute()
}
