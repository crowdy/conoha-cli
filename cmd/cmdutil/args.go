package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ExactArgs returns a PositionalArgs that reports the command's Use line on mismatch.
func ExactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("requires %d arg(s), received %d\n\nUsage:\n  %s", n, len(args), cmd.UseLine())
		}
		return nil
	}
}
