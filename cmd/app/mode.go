package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

type Mode string

const (
	ModeProxy   Mode = "proxy"
	ModeNoProxy Mode = "no-proxy"
)

var (
	ErrNoMarker     = errors.New("no mode marker on server")
	ErrModeConflict = errors.New("mode conflict")
)

// ParseMarker accepts the raw marker file content and returns the Mode.
func ParseMarker(s string) (Mode, error) {
	v := strings.TrimSpace(s)
	switch v {
	case string(ModeProxy):
		return ModeProxy, nil
	case string(ModeNoProxy):
		return ModeNoProxy, nil
	case "":
		return "", fmt.Errorf("empty marker")
	default:
		return "", fmt.Errorf("unknown marker value %q", v)
	}
}

// buildReadMarkerCmd prints marker contents or "__MISSING__" if absent.
// The distinct sentinel lets ReadMarker tell "file absent" apart from
// permission or SSH errors without relying on exit codes.
func buildReadMarkerCmd(app string) string {
	return fmt.Sprintf(
		`cat '/opt/conoha/%s/.conoha-mode' 2>/dev/null || echo __MISSING__`,
		app)
}

// buildWriteMarkerCmd creates the app dir (if missing) and writes the marker.
func buildWriteMarkerCmd(app string, m Mode) string {
	return fmt.Sprintf(
		`mkdir -p '/opt/conoha/%s' && printf %%s\\n '%s' > '/opt/conoha/%s/.conoha-mode'`,
		app, string(m), app)
}

// buildReadCurrentSlotCmd prints the active slot ID or empty output on absence.
func buildReadCurrentSlotCmd(app string) string {
	return fmt.Sprintf(
		`cat '/opt/conoha/%s/CURRENT_SLOT' 2>/dev/null || true`,
		app)
}

// formatModeConflictError returns a user-facing error wrapping ErrModeConflict.
// serverID is substituted into the recovery command hint so users can copy-run.
// When serverID is empty the literal "<server>" placeholder is kept (used when
// the caller doesn't know the server ID at error-construction time).
func formatModeConflictError(app, serverID string, got, want Mode) error {
	oppositeInit := "conoha app init"
	if want == ModeNoProxy {
		oppositeInit = "conoha app init --no-proxy"
	}
	server := serverID
	if server == "" {
		server = "<server>"
	}
	return fmt.Errorf(
		`app %q is initialized in %s mode on this server, but --%s was requested.
To switch modes:
    conoha app destroy %s               # removes the existing deployment
    %s %s       # re-initialize in %s mode
%w`,
		app, string(got), string(want), server, oppositeInit, server, string(want), ErrModeConflict)
}

// notInitializedError is the canonical error for a command that needs the
// app's mode marker but finds it missing. The recovery hint includes the right
// init subcommand based on mode when known; when mode is unset it suggests
// the bare 'conoha app init' form.
func notInitializedError(app, serverID string, mode Mode) error {
	server := serverID
	if server == "" {
		server = "<server>"
	}
	switch mode {
	case ModeNoProxy:
		return fmt.Errorf("app %q not initialized on this server — run 'conoha app init --no-proxy --app-name %s %s' first", app, app, server)
	case ModeProxy:
		return fmt.Errorf("app %q not initialized on this server — run 'conoha app init %s' first", app, server)
	default:
		return fmt.Errorf("app %q not initialized on this server — run 'conoha app init %s' first", app, server)
	}
}

// notDeployedError is the canonical error for "marker present but CURRENT_SLOT
// absent" — the app was initialized but never deployed.
func notDeployedError(app, serverID string) error {
	server := serverID
	if server == "" {
		server = "<server>"
	}
	return fmt.Errorf("app %q has not been deployed on this server — run 'conoha app deploy %s' first", app, server)
}

// ReadMarker returns the mode recorded on the server for app, or ErrNoMarker
// if no marker file exists.
func ReadMarker(cli *ssh.Client, app string) (Mode, error) {
	var buf bytes.Buffer
	if _, err := internalssh.RunCommand(cli, buildReadMarkerCmd(app), &buf, os.Stderr); err != nil {
		return "", fmt.Errorf("read marker: %w", err)
	}
	out := strings.TrimSpace(buf.String())
	if out == "__MISSING__" {
		return "", ErrNoMarker
	}
	return ParseMarker(out)
}

// WriteMarker persists the marker file on the server.
func WriteMarker(cli *ssh.Client, app string, m Mode) error {
	code, err := internalssh.RunCommand(cli, buildWriteMarkerCmd(app, m), os.Stderr, os.Stderr)
	if err != nil {
		return fmt.Errorf("write marker: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("write marker: exit %d", code)
	}
	return nil
}

// ReadCurrentSlot returns the active slot ID or "" when the file is absent.
// The returned value is re-validated via ValidateSlotID so a compromised or
// manually-edited CURRENT_SLOT cannot leak shell metacharacters into downstream
// 'docker compose -p <app>-<slot>' interpolation.
func ReadCurrentSlot(cli *ssh.Client, app string) (string, error) {
	var buf bytes.Buffer
	if _, err := internalssh.RunCommand(cli, buildReadCurrentSlotCmd(app), &buf, os.Stderr); err != nil {
		return "", fmt.Errorf("read CURRENT_SLOT: %w", err)
	}
	slot := strings.TrimSpace(buf.String())
	if slot == "" {
		return "", nil
	}
	if err := ValidateSlotID(slot); err != nil {
		return "", fmt.Errorf("CURRENT_SLOT: %w", err)
	}
	return slot, nil
}

// flagMode reads --proxy / --no-proxy flags and returns the intended mode, or
// "" if neither is set. Callers should have registered the flags mutually
// exclusive via AddModeFlags.
func flagMode(cmd *cobra.Command) Mode {
	if cmd.Flags().Lookup("no-proxy") != nil {
		if v, _ := cmd.Flags().GetBool("no-proxy"); v {
			return ModeNoProxy
		}
	}
	if cmd.Flags().Lookup("proxy") != nil {
		if v, _ := cmd.Flags().GetBool("proxy"); v {
			return ModeProxy
		}
	}
	return ""
}

// ResolveMode interprets flags against the marker.
// Precedence: flag override compared to marker (error on mismatch) > marker > ErrNoMarker.
// serverID is embedded into ErrModeConflict's recovery hint; pass "" if the
// caller does not have it.
func ResolveMode(cmd *cobra.Command, cli *ssh.Client, app, serverID string) (Mode, error) {
	want := flagMode(cmd)
	got, readErr := ReadMarker(cli, app)
	return resolveModeLogic(app, serverID, want, got, readErr)
}

// resolveModeLogic is the pure precedence layer extracted for unit testing.
// want is the flag-requested mode ("" if none). got/readErr come from ReadMarker.
// Non-ErrNoMarker read errors are propagated unchanged.
func resolveModeLogic(app, serverID string, want, got Mode, readErr error) (Mode, error) {
	if readErr != nil && !errors.Is(readErr, ErrNoMarker) {
		return "", readErr
	}
	switch {
	case want == "" && errors.Is(readErr, ErrNoMarker):
		return "", ErrNoMarker
	case want == "":
		return got, nil
	case errors.Is(readErr, ErrNoMarker):
		return want, nil
	case want != got:
		return "", formatModeConflictError(app, serverID, got, want)
	default:
		return got, nil
	}
}

// AddModeFlags registers --proxy and --no-proxy as mutually exclusive bool flags.
func AddModeFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("proxy", false, "force proxy (blue/green) mode, overriding server marker")
	cmd.Flags().Bool("no-proxy", false, "force no-proxy (flat single-slot) mode, overriding server marker")
	cmd.MarkFlagsMutuallyExclusive("proxy", "no-proxy")
}
