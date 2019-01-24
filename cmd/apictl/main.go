package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
)

var apictl = &cobra.Command{
	Use: "apictl [command]",
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func init() {
	keycheck := licensekeys.InitializeCommandFlags(apictl.PersistentFlags(), "apictl", Version)
	apictl.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true // https://github.com/spf13/cobra/issues/340
		err := keycheck(cmd.PersistentFlags())
		if err == nil {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	apictl.Version = Version
}

func recoverFromCrash() {
	if r := recover(); r != nil {
		fmt.Println("---")
		fmt.Println("\nThe apictl command has crashed. Sorry about that!")
		fmt.Println(r)
	}
}

func main() {
	defer recoverFromCrash()
	err := apictl.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func die(err error, args ...interface{}) {
	if err != nil {
		if args != nil {
			fmt.Printf("%v: %v\n", err, args)
		} else {
			fmt.Println(err)
		}
		panic(err)
	}
}
