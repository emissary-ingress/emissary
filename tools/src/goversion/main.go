package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// FlagErrorFunc is a function to be passed to (*cobra.Command).SetFlagErrorFunc that establishes
// GNU-ish behavior for invalid flag usage.
//
// If there is an error, FlagErrorFunc calls os.Exit; it does NOT return.  This means that all
// errors returned from (*cobra.Command).Execute will be execution errors, not usage errors.
func FlagErrorFunc(cmd *cobra.Command, err error) error {
	// Copyright note: This code was originally written by LukeShu for Telepresence.
	if err == nil {
		return nil
	}

	// If the error is multiple lines, include an extra blank line before the "See --help" line.
	errStr := strings.TrimRight(err.Error(), "\n")
	if strings.Contains(errStr, "\n") {
		errStr += "\n"
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\nSee '%s --help' for more information.\n", cmd.CommandPath(), errStr, cmd.CommandPath())
	os.Exit(2)
	return nil
}

func main() {
	ctx := context.Background()

	argparser := &cobra.Command{
		Use:   "goversion [flags] [COMMITISH]",
		Short: "Like `git describe`, but emits Go pseudo-versions",
		Args:  cobra.RangeArgs(0, 1),

		SilenceErrors: true, // main() will handle this after .ExecuteContext() returns
		SilenceUsage:  true, // our FlagErrorFunc will handle it
	}

	var argDirPrefix string
	argparser.Flags().StringVar(&argDirPrefix, "dir-prefix", "",
		"Consider the Go module `${COMMITISH}:${dir_prefix}/go.mod` instead of `${COMMITISH}:go.mod`")

	argparser.SetFlagErrorFunc(FlagErrorFunc)

	argparser.RunE = func(cmd *cobra.Command, args []string) error {
		commitish := "HEAD"
		if len(args) == 1 {
			commitish = args[0]
		}

		dirtyMarker := ""
		if commitish == "HEAD" {
			dirtyMarker = fmt.Sprintf("-dirty.%d", time.Now().Unix())
		}

		desc, err := Describe(cmd.Context(), commitish, argDirPrefix, dirtyMarker)
		if err != nil {
			return err
		}

		if dirtyMarker != "" && strings.HasPrefix(desc, dirtyMarker) && os.Getenv("CI") != "" {
			fmt.Fprintln(os.Stderr, "error: this should not happen in CI: the tree should not be dirty")
			// Don't bother checking for errors from .Run(), since these are
			// just informative error messages.
			cmd := exec.CommandContext(ctx, "git", "add", ".")
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
			cmd = exec.CommandContext(ctx, "git", "diff", "--cached")
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			cmd.Env = append(os.Environ(),
				"PAGER=")
			_ = cmd.Run()
			os.Exit(1)
		}

		fmt.Println(desc)
		return nil
	}

	if err := argparser.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(argparser.ErrOrStderr(), "%s: error: %v\n", argparser.CommandPath(), err)
		os.Exit(1)
	}
}
