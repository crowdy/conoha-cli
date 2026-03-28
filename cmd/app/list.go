package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	listCmd.Flags().StringP("user", "l", "root", "SSH user")
	listCmd.Flags().StringP("port", "p", "22", "SSH port")
	listCmd.Flags().StringP("identity", "i", "", "SSH private key path")
}

var listCmd = &cobra.Command{
	Use:   "list <id|name>",
	Short: "List deployed apps on a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)

		s, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		ip, err := internalssh.ServerIP(s)
		if err != nil {
			return err
		}

		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetString("port")
		identity, _ := cmd.Flags().GetString("identity")

		if identity == "" {
			identity = internalssh.ResolveKeyPath(s.KeyName)
		}
		if identity == "" {
			return fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
		}

		sshClient, err := internalssh.Connect(internalssh.ConnectConfig{
			Host:    ip,
			Port:    port,
			User:    user,
			KeyPath: identity,
		})
		if err != nil {
			return fmt.Errorf("SSH connect: %w", err)
		}
		defer func() { _ = sshClient.Close() }()

		script := generateListScript()
		exitCode, err := internalssh.RunScript(sshClient, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("list failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("list exited with code %d", exitCode)
		}
		return nil
	},
}

func generateListScript() []byte {
	return []byte(`#!/bin/bash
for repo in /opt/conoha/*.git; do
    [ -d "$repo" ] || continue
    APP_NAME=$(basename "$repo" .git)
    WORK_DIR="/opt/conoha/${APP_NAME}"

    if [ -d "$WORK_DIR" ] && (cd "$WORK_DIR" && docker compose ps --status running -q 2>/dev/null | grep -q .); then
        STATUS="running"
    elif [ -d "$WORK_DIR" ] && (cd "$WORK_DIR" && docker compose ps -q 2>/dev/null | grep -q .); then
        STATUS="stopped"
    else
        STATUS="no containers"
    fi

    printf "%-30s %s\n" "$APP_NAME" "$STATUS"
done
`)
}
