package runner

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
)

var argparser = &cobra.Command{}

func Main(version string) {
	keycheck := licensekeys.InitializeCommandFlags(argparser.PersistentFlags(), "ambassador-sidecar", version)

	argparser.Use = os.Args[0]
	argparser.Version = version
	argparser.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// https://github.com/spf13/cobra/issues/340
		cmd.SilenceUsage = true

		// License key validation
		err := keycheck(cmd.PersistentFlags())
		if err == nil {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		time.Sleep(5 * 60 * time.Second)
		os.Exit(1)
	}

	err := argparser.Execute()
	if err != nil {
		os.Exit(1)
	}
}
