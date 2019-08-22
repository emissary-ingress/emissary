package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	argparser.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show the program's version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	})
}
