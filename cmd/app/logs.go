package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(logsCmd)
	logsCmd.Flags().BoolP("follow", "f", false, "stream logs in real-time")
	logsCmd.Flags().Int("tail", 100, "number of lines to show")
	logsCmd.Flags().String("service", "", "specific service name")
	AddModeFlags(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs <id|name>",
	Short: "Show app container logs",
	Long:  "Show docker compose logs for the active slot (proxy mode) or the flat work dir (no-proxy). Use --follow to stream in real-time (Ctrl+C to stop).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetInt("tail")
		service, _ := cmd.Flags().GetString("service")
		if service != "" {
			if err := internalssh.ValidateAppName(service); err != nil {
				return fmt.Errorf("invalid service name: %w", err)
			}
		}

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
			composeCmd = buildLogsCmdForProxy(ctx.AppName, slot, tail, follow, service)
		} else {
			composeCmd = buildLogsCmdForNoProxy(ctx.AppName, tail, follow, service)
		}

		exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("logs failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("logs exited with code %d", exitCode)
		}
		return nil
	},
}

func buildLogsCmdForProxy(app, slot string, tail int, follow bool, service string) string {
	cmd := fmt.Sprintf("docker compose -p %s-%s logs --tail %d", app, slot, tail)
	if follow {
		cmd += " -f"
	}
	if service != "" {
		cmd += " " + service
	}
	return cmd
}

func buildLogsCmdForNoProxy(app string, tail int, follow bool, service string) string {
	cmd := fmt.Sprintf("cd /opt/conoha/%s && docker compose logs --tail %d", app, tail)
	if follow {
		cmd += " -f"
	}
	if service != "" {
		cmd += " " + service
	}
	return cmd
}
