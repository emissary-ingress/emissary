package main

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/rls"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

var watch = &cobra.Command{
	Use:   "rls-watch",
	Short: "Watch RateLimit CRD files",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := types.WrapLogrus(logrus.New())
		cfg, errs := types.ConfigFromEnv()
		for _, err := range errs {
			// This is only fatal if cfg.Output == ""
			// (see below)
			logger.Errorln("config error:", err)
		}
		if cfg.Output == "" {
			return errs[len(errs)-1]
		}
		return rls.DoWatch(context.Background(), cfg, logger)
	},
}

func init() {
	argparser.AddCommand(watch)
}
