package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = &cobra.Command{
	Use:   "version",
	Short: "Show the program's version number",
	Run:   showVersion,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

func init() {
	apictl.AddCommand(version)
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Println(Version)
}
