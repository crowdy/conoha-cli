package app

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// envFilePath returns the canonical server-side env file path for app.
// v0.2+ location: app-dir local dotfile so it rides along with the rest
// of the app state under /opt/conoha/<name>/ and gets wiped cleanly by
// `rm -rf /opt/conoha/<name>` on destroy.
func envFilePath(app string) string {
	return fmt.Sprintf("/opt/conoha/%s/.env.server", app)
}

// legacyEnvFilePath returns the v0.1.x env file location — one level up from
// the app dir. Still read on-the-fly for backward compat until v0.3.x; the
// `app env migrate` subcommand copies content to the new location.
func legacyEnvFilePath(app string) string {
	return fmt.Sprintf("/opt/conoha/%s.env.server", app)
}

// effectiveEnvFileScript renders a bash snippet that sets ENV_FILE to the
// first existing file between the new and legacy paths, falling back to the
// new path (+ a deprecation warning on stderr) when only the legacy one
// exists. Every env subcommand uses this to stay read-compatible with old
// servers without auto-migrating content.
func effectiveEnvFileScript(app string) string {
	return fmt.Sprintf(
		`NEW_ENV="%s"
LEGACY_ENV="%s"
if [ ! -f "$NEW_ENV" ] && [ -f "$LEGACY_ENV" ]; then
    echo "warning: reading legacy env file $LEGACY_ENV (run 'conoha app env migrate <server>' to move it to $NEW_ENV; legacy path will stop being read in v0.3)" >&2
    ENV_FILE="$LEGACY_ENV"
else
    ENV_FILE="$NEW_ENV"
fi
`, envFilePath(app), legacyEnvFilePath(app))
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
	envCmd.AddCommand(envMigrateCmd)

	addAppFlags(envSetCmd)
	addAppFlags(envGetCmd)
	addAppFlags(envListCmd)
	addAppFlags(envUnsetCmd)
	addAppFlags(envMigrateCmd)
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
		fmt.Fprintf(os.Stderr, "Next: run 'conoha app deploy %s' to apply.\n", ctx.ServerID)
		return nil
	},
}

// legacyOnlyGuardScript aborts the calling script when only the legacy env
// file exists. Without this guard, writing to $NEW_ENV would create a second
// file, and the deploy-time "new wins if present" logic would silently hide
// any KEY=VALUE previously stored in legacy — unrecoverable without backups.
func legacyOnlyGuardScript(appName string) string {
	return fmt.Sprintf(
		`NEW_ENV=%q
LEGACY_ENV=%q
if [ ! -f "$NEW_ENV" ] && [ -f "$LEGACY_ENV" ]; then
    echo "error: legacy env file $LEGACY_ENV exists but $NEW_ENV does not." >&2
    echo "Writing here would silently hide legacy values on next deploy." >&2
    echo "Run 'conoha app env migrate <server>' first, then retry." >&2
    exit 2
fi
`, envFilePath(appName), legacyEnvFilePath(appName))
}

func generateEnvSetScript(appName string, env map[string]string) []byte {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -euo pipefail\n")
	// Guard against legacy-only state (data-loss prevention). Then write to
	// the new canonical location only — migrate is the explicit path for
	// legacy content movement.
	b.WriteString(legacyOnlyGuardScript(appName))
	fmt.Fprintf(&b, "mkdir -p '/opt/conoha/%s'\n", appName)
	fmt.Fprintf(&b, "ENV_FILE=%q\n", envFilePath(appName))
	b.WriteString("touch \"$ENV_FILE\"\n")
	// Enforce 0600 on every write; `touch` honors umask (usually 0644) and
	// `mv` preserves the source mode, neither of which is appropriate for a
	// file that may contain credentials.
	b.WriteString("chmod 600 \"$ENV_FILE\"\n")

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
		b.WriteString("chmod 600 \"$ENV_FILE\"\n")
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
	return fmt.Sprintf("bash -c '%s grep \"^%s=\" \"$ENV_FILE\" | cut -d= -f2-'",
		strings.ReplaceAll(effectiveEnvFileScript(appName), "'", "'\\''"),
		key)
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

		command := generateEnvListCommand(ctx.AppName)
		_, err = internalssh.RunCommand(ctx.Client, command, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env list failed: %w", err)
		}
		return nil
	},
}

func generateEnvListCommand(appName string) string {
	return fmt.Sprintf("bash -c '%s cat \"$ENV_FILE\" 2>/dev/null || true'",
		strings.ReplaceAll(effectiveEnvFileScript(appName), "'", "'\\''"))
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
		fmt.Fprintf(os.Stderr, "Next: run 'conoha app deploy %s' to apply.\n", ctx.ServerID)
		return nil
	},
}

func generateEnvUnsetScript(appName string, keys []string) []byte {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -euo pipefail\n")
	// Symmetric with set: refuse on legacy-only state so unset can't silently
	// act on a file that will be hidden at deploy time.
	b.WriteString(legacyOnlyGuardScript(appName))
	b.WriteString(effectiveEnvFileScript(appName))
	b.WriteString("[ -f \"$ENV_FILE\" ] || exit 0\n")
	b.WriteString("cp \"$ENV_FILE\" \"$ENV_FILE.tmp\"\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "grep -v \"^%s=\" \"$ENV_FILE.tmp\" > \"$ENV_FILE.tmp2\" || true\n", k)
		b.WriteString("mv \"$ENV_FILE.tmp2\" \"$ENV_FILE.tmp\"\n")
	}
	b.WriteString("mv \"$ENV_FILE.tmp\" \"$ENV_FILE\"\n")
	b.WriteString("chmod 600 \"$ENV_FILE\"\n")
	return []byte(b.String())
}

var envMigrateCmd = &cobra.Command{
	Use:   "migrate <server>",
	Short: "Move legacy /opt/conoha/<app>.env.server to /opt/conoha/<app>/.env.server",
	Long: `One-time operator step: relocates the v0.1.x env file at the legacy path
(one level up from the app work dir) to the v0.2+ canonical location
(dotfile inside the app work dir). Safe to re-run — a no-op when the
legacy path is already absent.

Fails with a clear error when both old and new locations exist, to avoid
silently overwriting either.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args[:1])
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		script := generateEnvMigrateScript(ctx.AppName)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("env migrate failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("env migrate exited with code %d", exitCode)
		}
		return nil
	},
}

func generateEnvMigrateScript(appName string) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

NEW_ENV=%q
LEGACY_ENV=%q

if [ ! -f "$LEGACY_ENV" ]; then
    echo "Nothing to migrate: $LEGACY_ENV does not exist."
    exit 0
fi
if [ -f "$NEW_ENV" ]; then
    echo "Both $NEW_ENV and $LEGACY_ENV exist." >&2
    echo "Resolve manually: merge and delete one." >&2
    exit 1
fi

mkdir -p "$(dirname "$NEW_ENV")"
mv "$LEGACY_ENV" "$NEW_ENV"
chmod 600 "$NEW_ENV"
echo "Migrated: $LEGACY_ENV -> $NEW_ENV"
`, envFilePath(appName), legacyEnvFilePath(appName)))
}
