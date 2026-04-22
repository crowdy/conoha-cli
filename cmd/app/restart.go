package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(restartCmd)
	AddModeFlags(restartCmd)
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

		mode, err := ResolveModeFromCtx(cmd, ctx)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return notInitializedError(ctx.AppName, ctx.ServerID, "")
			}
			return err
		}

		var composeCmd string
		if mode == ModeProxy {
			slot, err := ReadCurrentSlot(ctx.Client, ctx.AppName)
			if err != nil {
				return err
			}
			if slot == "" {
				return notDeployedError(ctx.AppName, ctx.ServerID)
			}
			composeCmd = buildRestartCmdForProxy(ctx.AppName, slot)
		} else {
			composeCmd = buildRestartCmdForNoProxy(ctx.AppName)
		}

		fmt.Fprintf(os.Stderr, "Restarting app %q on %s...\n", ctx.AppName, ctx.Server.Name)
		exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("restart failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("restart exited with code %d", exitCode)
		}
		return nil
	},
}

func buildRestartCmdForProxy(app, slot string) string {
	return fmt.Sprintf("docker compose -p %s-%s restart && docker compose -p %s-%s ps", app, slot, app, slot)
}

func buildRestartCmdForNoProxy(app string) string {
	return fmt.Sprintf("cd /opt/conoha/%s && docker compose restart && docker compose ps", app)
}
