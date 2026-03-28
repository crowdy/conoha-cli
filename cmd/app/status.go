package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status <id|name>",
	Short: "Show app container status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		workDir := "/opt/conoha/" + ctx.AppName
		exitCode, err := internalssh.RunCommand(ctx.Client, fmt.Sprintf("cd %s && docker compose ps", workDir), os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("status failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("status exited with code %d", exitCode)
		}
		return nil
	},
}
