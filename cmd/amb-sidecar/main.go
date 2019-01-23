package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

var argparser = &cobra.Command{}

func main() {
	keycheck := licensekeys.InitializeCommandFlags(argparser.PersistentFlags(), Version)

	argparser.Use = os.Args[0]
	argparser.Version = Version
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
