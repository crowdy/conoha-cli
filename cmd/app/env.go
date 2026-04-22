package app

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// proxyEnvWarningMessage returns the one-line warning emitted when `app env`
// is run against a proxy-mode app. See #94 for the planned redesign.
func proxyEnvWarningMessage() string {
	return "warning: app env has no effect on proxy-mode deployed slots; see #94 for the redesign\n"
}

// maybeWarnProxyEnvMode emits the proxy-mode warning to stderr once per env
// subcommand invocation. Silent on no-proxy or when marker lookup fails.
// Consumes the ctx-cached marker so read-only subcommands (get, list) don't
// pay a second SSH round-trip alongside the env file read.
func maybeWarnProxyEnvMode(ctx *appContext) {
	m, err := ctx.Marker()
	if err == nil && m == ModeProxy {
		fmt.Fprint(os.Stderr, proxyEnvWarningMessage())
	}
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage app environment variables",
}

func init() {
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envGetCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envUnsetCmd)

	addAppFlags(envSetCmd)
	addAppFlags(envGetCmd)
	addAppFlags(envListCmd)
	addAppFlags(envUnsetCmd)
}

var envSetCmd = &cobra.Command{
	Use:   "set <server> KEY=VALUE [KEY=VALUE...]",
	Short: "Set environment variables",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args[:1])
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		maybeWarnProxyEnvMode(ctx)

		env := make(map[string]string)
		for _, arg := range args[1:] {
			k, v, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid format %q (expected KEY=VALUE)", arg)
			}
			if err := internalssh.ValidateEnvKey(k); err != nil {
				return err
			}
			env[k] = v
		}

		script := generateEnvSetScript(ctx.AppName, env)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env set failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("env set exited with code %d", exitCode)
		}

		keys := make([]string, 0, len(env))
		for k := range env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(os.Stderr, "Set %s\n", k)
		}
		return nil
	},
}

func generateEnvSetScript(appName string, env map[string]string) []byte {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -euo pipefail\n")
	fmt.Fprintf(&b, "ENV_FILE=\"/opt/conoha/%s.env.server\"\n", appName)
	b.WriteString("touch \"$ENV_FILE\"\n")

	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := env[k]
		fmt.Fprintf(&b, "grep -v \"^%s=\" \"$ENV_FILE\" > \"$ENV_FILE.tmp\" || true\n", k)
		escaped := strings.ReplaceAll(v, "'", "'\\''")
		fmt.Fprintf(&b, "echo '%s=%s' >> \"$ENV_FILE.tmp\"\n", k, escaped)
		b.WriteString("mv \"$ENV_FILE.tmp\" \"$ENV_FILE\"\n")
	}
	return []byte(b.String())
}

var envGetCmd = &cobra.Command{
	Use:   "get <server> KEY",
	Short: "Get an environment variable value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args[:1])
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		maybeWarnProxyEnvMode(ctx)

		key := args[1]
		if err := internalssh.ValidateEnvKey(key); err != nil {
			return err
		}

		command := generateEnvGetCommand(ctx.AppName, key)
		exitCode, err := internalssh.RunCommand(ctx.Client, command, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env get failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("environment variable %q not set", key)
		}
		return nil
	},
}

func generateEnvGetCommand(appName, key string) string {
	return fmt.Sprintf(
		`grep "^%s=" /opt/conoha/%s.env.server | cut -d= -f2-`,
		key, appName)
}

var envListCmd = &cobra.Command{
	Use:   "list <server>",
	Short: "List all environment variables",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		maybeWarnProxyEnvMode(ctx)

		command := generateEnvListCommand(ctx.AppName)
		_, err = internalssh.RunCommand(ctx.Client, command, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env list failed: %w", err)
		}
		return nil
	},
}

func generateEnvListCommand(appName string) string {
	return fmt.Sprintf(`cat /opt/conoha/%s.env.server 2>/dev/null || true`, appName)
}

var envUnsetCmd = &cobra.Command{
	Use:   "unset <server> KEY [KEY...]",
	Short: "Remove environment variables",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args[:1])
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		maybeWarnProxyEnvMode(ctx)

		keys := args[1:]
		for _, k := range keys {
			if err := internalssh.ValidateEnvKey(k); err != nil {
				return err
			}
		}

		script := generateEnvUnsetScript(ctx.AppName, keys)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env unset failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("env unset exited with code %d", exitCode)
		}

		for _, k := range keys {
			fmt.Fprintf(os.Stderr, "Unset %s\n", k)
		}
		return nil
	},
}

func generateEnvUnsetScript(appName string, keys []string) []byte {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -euo pipefail\n")
	fmt.Fprintf(&b, "ENV_FILE=\"/opt/conoha/%s.env.server\"\n", appName)
	b.WriteString("[ -f \"$ENV_FILE\" ] || exit 0\n")
	b.WriteString("cp \"$ENV_FILE\" \"$ENV_FILE.tmp\"\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "grep -v \"^%s=\" \"$ENV_FILE.tmp\" > \"$ENV_FILE.tmp2\" || true\n", k)
		b.WriteString("mv \"$ENV_FILE.tmp2\" \"$ENV_FILE.tmp\"\n")
	}
	b.WriteString("mv \"$ENV_FILE.tmp\" \"$ENV_FILE\"\n")
	return []byte(b.String())
}
