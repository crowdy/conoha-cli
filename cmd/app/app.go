package app

import "github.com/spf13/cobra"

// Cmd is the app command group.
var Cmd = &cobra.Command{
	Use:   "app",
	Short: "Application deployment commands",
}

func init() {
	Cmd.AddCommand(initCmd)
}
