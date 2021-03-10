package reproducer

import (
	"context"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:  "reproducer [command]",
	Long: `The reproducer command is used to extract debug info and enough ambassador inputs to create high fidelity reproducers of issues encountered with ambassador edge-stack. See the help for each subcommands for more details.`,
}

func init() {
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(createCmd)
}

func Main(ctx context.Context, version string, args ...string) error {
	return rootCmd.ExecuteContext(ctx)
}
