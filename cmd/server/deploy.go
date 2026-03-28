package server

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	deployCmd.Flags().String("script", "", "local script file to upload and execute")
	deployCmd.Flags().StringArray("env", nil, "environment variables (KEY=VALUE, repeatable)")
	deployCmd.Flags().StringP("user", "l", "root", "SSH user")
	deployCmd.Flags().StringP("port", "p", "22", "SSH port")
	deployCmd.Flags().StringP("identity", "i", "", "SSH private key path")
	_ = deployCmd.MarkFlagRequired("script")
}

var deployCmd = &cobra.Command{
	Use:   "deploy <id|name>",
	Short: "Deploy a script to a server via SSH",
	Long:  "Upload and execute a local script on a remote server. Streams output in real-time.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}

		s, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		ip, err := internalssh.ServerIP(s)
		if err != nil {
			return err
		}

		scriptPath, _ := cmd.Flags().GetString("script")
		envFlags, _ := cmd.Flags().GetStringArray("env")
		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetString("port")
		identity, _ := cmd.Flags().GetString("identity")

		if identity == "" {
			identity = internalssh.ResolveKeyPath(s.KeyName)
		}
		if identity == "" {
			return fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
		}

		script, err := os.ReadFile(scriptPath)
		if err != nil {
			return fmt.Errorf("read script %s: %w", scriptPath, err)
		}

		env := make(map[string]string)
		for _, e := range envFlags {
			k, v, ok := strings.Cut(e, "=")
			if !ok {
				return fmt.Errorf("invalid --env format %q (expected KEY=VALUE)", e)
			}
			if err := internalssh.ValidateEnvKey(k); err != nil {
				return err
			}
			env[k] = v
		}

		client, err := internalssh.Connect(internalssh.ConnectConfig{
			Host:    ip,
			Port:    port,
			User:    user,
			KeyPath: identity,
		})
		if err != nil {
			return fmt.Errorf("SSH connect: %w", err)
		}
		defer func() { _ = client.Close() }()

		fmt.Fprintf(os.Stderr, "Deploying %s to %s (%s)...\n", scriptPath, s.Name, ip)

		exitCode, err := internalssh.RunScript(client, script, env, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("deploy failed: %w", err)
		}

		if exitCode != 0 {
			return fmt.Errorf("script exited with code %d", exitCode)
		}

		fmt.Fprintf(os.Stderr, "Deploy complete.\n")
		return nil
	},
}
