package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
)

var apictl = &cobra.Command{
	Use: "apictl [command]",
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

var LICENSE_PAUSE = map[*cobra.Command]bool{
	watch: true,
}

func init() {
	keycheck := licensekeys.InitializeCommandFlags(apictl.PersistentFlags(), Version)
	apictl.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		err := keycheck(cmd.PersistentFlags())
		if err == nil {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		if pause, _ := LICENSE_PAUSE[cmd]; pause {
			time.Sleep(5 * 60 * time.Second)
		}
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
	apictl.Execute()
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
