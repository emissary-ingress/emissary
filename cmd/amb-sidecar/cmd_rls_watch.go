package main

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/rls"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func init() {
	argparser.AddCommand(&cobra.Command{
		Use:   "rls-watch",
		Short: "Watch RateLimit CRD files",
		RunE: func(cmd *cobra.Command, args []string) error {
			l := types.WrapLogrus(logrus.New())

			cfg, warn, fatal := types.ConfigFromEnv()
			for _, err := range warn {
				l.Warnln("config error:", err)
			}
			for _, err := range fatal {
				l.Errorln("config error:", err)
			}
			if len(fatal) > 0 {
				return fatal[len(fatal)-1]
			}

			return rls.DoWatch(context.Background(), cfg, l)
		},
	})
}
