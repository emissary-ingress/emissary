package main

import (
	"fmt"
	"os"
)

func main() {
	args, err := ParseArgs(os.Args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		fmt.Fprintf(os.Stderr, "Usage: %s TARGET [INPUTFILES...]\n", os.Args[0])
		os.Exit(2)
	}
	if err := Main(args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}
