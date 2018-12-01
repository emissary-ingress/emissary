package main

import (
	"github.com/spf13/cobra"
)

var apictl = &cobra.Command{Use: "apictl [command]"}

func main() {
	apictl.Execute()
}
