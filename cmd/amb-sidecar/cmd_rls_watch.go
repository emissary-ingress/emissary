package main

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/rls"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

var output string

var watch = &cobra.Command{
	Use:   "rls-watch",
	Short: "Watch RateLimit CRD files",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := types.Config{
			AmbassadorID:              os.Getenv("AMBASSADOR_ID"),
			AmbassadorNamespace:       os.Getenv("AMBASSADOR_NAMESPACE"),
			AmbassadorSingleNamespace: os.Getenv("AMBASSADOR_SINGLE_NAMESPACE") != "",
			Output:                    output,
		}
		if cfg.AmbassadorID == "" {
			cfg.AmbassadorID = "default"
		}
		if cfg.AmbassadorNamespace == "" {
			cfg.AmbassadorNamespace = "default"
		}
		return rls.DoWatch(context.Background(), cfg, types.WrapLogrus(logrus.New()))
	},
}

func init() {
	argparser.AddCommand(watch)
	watch.Flags().StringVarP(&output, "output", "o", "", "output directory")
	watch.MarkFlagRequired("output")
}
