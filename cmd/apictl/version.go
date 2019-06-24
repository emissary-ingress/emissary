package main

import (
	"text/template"

	"github.com/spf13/cobra"
)

func init() {
	apictl.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show the program's version number",
		RunE: func(cmd *cobra.Command, args []string) error {
			t := template.New("top")
			template.Must(t.Parse(cmd.VersionTemplate()))
			return t.Execute(cmd.OutOrStdout(), apictl)
		},

		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	})
}
