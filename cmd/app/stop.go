package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/prompt"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(stopCmd)
	AddModeFlags(stopCmd)
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

		// Resolve mode + slot before the prompt so flag/marker conflicts or
		// "not deployed" errors abort without asking the user to confirm (I3).
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
			composeCmd = buildStopCmdForProxy(ctx.AppName, slot)
		} else {
			composeCmd = buildStopCmdForNoProxy(ctx.AppName)
		}

		ok, err := prompt.Confirm(fmt.Sprintf("Stop app %q on %s?", ctx.AppName, ctx.Server.Name))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}

		fmt.Fprintf(os.Stderr, "Stopping app %q on %s...\n", ctx.AppName, ctx.Server.Name)
		exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("stop failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("stop exited with code %d", exitCode)
		}
		return nil
	},
}

func buildStopCmdForProxy(app, slot string) string {
	return fmt.Sprintf("docker compose -p %s-%s stop && docker compose -p %s-%s ps", app, slot, app, slot)
}

func buildStopCmdForNoProxy(app string) string {
	return fmt.Sprintf("cd /opt/conoha/%s && docker compose stop && docker compose ps", app)
}
