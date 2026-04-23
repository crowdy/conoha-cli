package app

import "github.com/spf13/cobra"

// Cmd is the app command group.
var Cmd = &cobra.Command{
	Use:   "app",
	Short: "Application deployment commands",
}

func init() {
	Cmd.AddCommand(initCmd)
	Cmd.AddCommand(deployCmd)
	Cmd.AddCommand(rollbackCmd)
	Cmd.AddCommand(logsCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(restartCmd)
	Cmd.AddCommand(envCmd)
	Cmd.AddCommand(destroyCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(resetCmd)
}
