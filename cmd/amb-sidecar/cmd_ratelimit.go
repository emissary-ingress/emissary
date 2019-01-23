package main

import (
	"github.com/spf13/cobra"

	"github.com/lyft/ratelimit/src/service_cmd/runner"
)

func init() {
	argparser.AddCommand(&cobra.Command{
		Use:   "ratelimit",
		Short: "Run the Lyft ratelimit service process",
		Run: func(cmd *cobra.Command, args []string) {
			runner.Run()
		},
	})
}
