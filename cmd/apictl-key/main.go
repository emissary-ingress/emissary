package main

import (
	"github.com/spf13/cobra"
)

var apictl_key = &cobra.Command{
	Use: "apictl-key [command]",
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	apictl_key.Execute()
}
