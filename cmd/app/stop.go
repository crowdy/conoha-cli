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

		ok, err := prompt.Confirm(fmt.Sprintf("Stop app %q on %s?", ctx.AppName, ctx.Server.Name))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}

		mode, err := ResolveMode(cmd, ctx.Client, ctx.AppName)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return fmt.Errorf("app %q has not been initialized on this server", ctx.AppName)
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
				return fmt.Errorf("app %q has not been deployed on this server", ctx.AppName)
			}
			composeCmd = buildStopCmdForProxy(ctx.AppName, slot)
		} else {
			composeCmd = buildStopCmdForNoProxy(ctx.AppName)
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
