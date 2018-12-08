package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apictl = &cobra.Command{Use: "apictl [command]"}

func main() {
	apictl.Execute()
}

func die(err error, args ...interface{}) {
	if err != nil {
		if args != nil {
			fmt.Printf("%v: %v\n", err, args)
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}
