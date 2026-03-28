package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/prompt"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop <id|name>",
	Short: "Stop app containers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		ok, err := prompt.Confirm(fmt.Sprintf("Stop app %q on %s?", ctx.AppName, ctx.Server.Name))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}

		workDir := "/opt/conoha/" + ctx.AppName
		fmt.Fprintf(os.Stderr, "Stopping app %q on %s...\n", ctx.AppName, ctx.Server.Name)
		exitCode, err := internalssh.RunCommand(ctx.Client, fmt.Sprintf("cd %s && docker compose stop && docker compose ps", workDir), os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("stop failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("stop exited with code %d", exitCode)
		}
		return nil
	},
}
