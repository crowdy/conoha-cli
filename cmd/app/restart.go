package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(restartCmd)
}

var restartCmd = &cobra.Command{
	Use:   "restart <id|name>",
	Short: "Restart app containers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		workDir := "/opt/conoha/" + ctx.AppName
		fmt.Fprintf(os.Stderr, "Restarting app %q on %s...\n", ctx.AppName, ctx.Server.Name)
		exitCode, err := internalssh.RunCommand(ctx.Client, fmt.Sprintf("cd %s && docker compose restart && docker compose ps", workDir), os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("restart failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("restart exited with code %d", exitCode)
		}
		return nil
	},
}
