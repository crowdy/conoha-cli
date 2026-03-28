package app

import (
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
}

var logsCmd = &cobra.Command{
	Use:   "logs <id|name>",
	Short: "Show app container logs",
	Long:  "Show docker compose logs. Use --follow to stream in real-time (Ctrl+C to stop).",
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

		workDir := "/opt/conoha/" + ctx.AppName
		composeCmd := fmt.Sprintf("cd %s && docker compose logs --tail %d", workDir, tail)
		if follow {
			composeCmd += " -f"
		}
		if service != "" {
			if err := internalssh.ValidateAppName(service); err != nil {
				return fmt.Errorf("invalid service name: %w", err)
			}
			composeCmd += " " + service
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
