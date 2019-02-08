package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/rls"
)

var output string

var watch = &cobra.Command{
	Use:   "rls-watch",
	Short: "Watch RateLimit CRD files",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rls.DoWatch(logrus.New(), output)
	},
}

func init() {
	argparser.AddCommand(watch)
	watch.Flags().StringVarP(&output, "output", "o", "", "output directory")
	watch.MarkFlagRequired("output")
}
