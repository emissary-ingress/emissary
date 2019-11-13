package main

import (
	"os"

	"github.com/spf13/cobra"
)

var argparser = &cobra.Command{
	Use: os.Args[0],
}

func main() {
	argparser.Execute()
}
