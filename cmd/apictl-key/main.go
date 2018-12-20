package main

import (
	"github.com/spf13/cobra"
)

var apictl_key = &cobra.Command{
	Use: "apictl-key [command]",
}

func main() {
	apictl_key.Execute()
}
