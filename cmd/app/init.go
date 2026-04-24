package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/config"
	"github.com/crowdy/conoha-cli/internal/model"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(initCmd)
	initCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	AddModeFlags(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init <server>",
	Short: "Register the app's conoha.yml with conoha-proxy on the server",
	Long: `Read conoha.yml from the current directory, verify the server has Docker
and a running conoha-proxy, and upsert the service (name, hosts, health policy)
against the proxy's Admin API.

Run 'conoha proxy boot' on the server first.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		noProxy, _ := cmd.Flags().GetBool("no-proxy")
		if noProxy {
			return runInitNoProxy(cmd, args[0])
		}
		return runInitProxy(cmd, args[0])
	},
}

func runInitProxy(cmd *cobra.Command, serverID string) error {
	pf, err := config.LoadProjectFile(config.ProjectFileName)
	if err != nil {
		return err
	}
	if err := pf.Validate(); err != nil {
		return err
	}
	composePath, err := pf.ResolveComposeFile(".")
	if err != nil {
		return err
	}
	if err := pf.ValidateAgainstCompose(composePath); err != nil {
		return err
	}

	sshClient, s, ip, err := connectToServer(cmd, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sshClient.Close() }()

	// Reject implicit mode switch — user must `app destroy` first.
	if existing, err := ReadMarker(sshClient, pf.Name); err == nil && existing != ModeProxy {
		return formatModeConflictError(pf.Name, serverID, existing, ModeProxy)
	} else if err != nil && !errors.Is(err, ErrNoMarker) {
		return err
	}

	dataDir, _ := cmd.Flags().GetString("data-dir")
	client := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))

	if err := warnOnLegacyRepo(sshClient, pf.Name); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "==> Registering service %q on %s (%s)\n", pf.Name, s.Name, ip)
	if err := registerProxyServices(client, pf, os.Stderr); err != nil {
		return err
	}
	if err := WriteMarker(sshClient, pf.Name, ModeProxy); err != nil {
		return fmt.Errorf("write mode marker: %w", err)
	}
	// Touch an empty .env.server so the compose override's env_file: entry
	// resolves on first deploy. Subsequent `app env set` grows the file.
	if err := touchEnvFile(sshClient, pf.Name); err != nil {
		return fmt.Errorf("touching env stub: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Next: run 'conoha app deploy %s' to push your app.\n", serverID)
	return nil
}

// touchEnvFile creates an empty /opt/conoha/<app>/.env.server if it doesn't
// already exist and enforces 0600 mode (file may hold credentials; `touch`
// alone honors umask, typically 0644). Idempotent; re-running app init is
// safe.
func touchEnvFile(cli *ssh.Client, app string) error {
	command := fmt.Sprintf(
		"mkdir -p '/opt/conoha/%s' && touch '/opt/conoha/%s/.env.server' && chmod 600 '/opt/conoha/%s/.env.server'",
		app, app, app,
	)
	code, err := internalssh.RunCommand(cli, command, os.Stderr, os.Stderr)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("touch env: exit %d", code)
	}
	return nil
}

func runInitNoProxy(cmd *cobra.Command, serverID string) error {
	appName, _ := cmd.Flags().GetString("app-name")
	if appName == "" {
		return fmt.Errorf("--app-name is required with --no-proxy")
	}
	if err := internalssh.ValidateAppName(appName); err != nil {
		return err
	}
	sshClient, s, ip, err := connectToServer(cmd, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sshClient.Close() }()

	// Reject implicit mode switch — user must `app destroy` first.
	if existing, err := ReadMarker(sshClient, appName); err == nil && existing != ModeNoProxy {
		return formatModeConflictError(appName, serverID, existing, ModeNoProxy)
	} else if err != nil && !errors.Is(err, ErrNoMarker) {
		return err
	}

	// Verify docker is present.
	code, err := internalssh.RunCommand(sshClient, "command -v docker >/dev/null 2>&1", os.Stderr, os.Stderr)
	if err != nil {
		return fmt.Errorf("docker check: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("docker is not installed on %s (%s)", s.Name, ip)
	}

	fmt.Fprintf(os.Stderr, "==> Initializing %q on %s (%s) in no-proxy mode\n", appName, s.Name, ip)
	if err := WriteMarker(sshClient, appName, ModeNoProxy); err != nil {
		return err
	}
	// Touch an empty .env.server so `app env set` / deploy merges always
	// have a well-known file to read (even on first deploy).
	if err := touchEnvFile(sshClient, appName); err != nil {
		return fmt.Errorf("touching env stub: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Initialized. Next: run 'conoha app deploy --no-proxy --app-name %s %s'\n", appName, serverID)
	return nil
}

// connectToServer opens an SSH session to the server identified by id-or-name.
// Returns the client, the resolved server, and its public IP.
func connectToServer(cmd *cobra.Command, idOrName string) (*ssh.Client, *model.Server, string, error) {
	apiClient, err := cmdutil.NewClient(cmd)
	if err != nil {
		return nil, nil, "", err
	}
	compute := api.NewComputeAPI(apiClient)
	s, err := compute.FindServer(idOrName)
	if err != nil {
		return nil, nil, "", err
	}
	ip, err := internalssh.ServerIP(s)
	if err != nil {
		return nil, nil, "", err
	}
	user, _ := cmd.Flags().GetString("user")
	port, _ := cmd.Flags().GetString("port")
	identity, _ := cmd.Flags().GetString("identity")
	if identity == "" {
		identity = internalssh.ResolveKeyPath(s.KeyName)
	}
	if identity == "" {
		return nil, nil, "", fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
	}
	cli, err := internalssh.Connect(internalssh.ConnectConfig{
		Host: ip, Port: port, User: user, KeyPath: identity,
	})
	if err != nil {
		return nil, nil, "", fmt.Errorf("SSH connect: %w", err)
	}
	return cli, s, ip, nil
}

// mapHealth copies project-file health settings into the proxy request shape.
func mapHealth(h *config.HealthSpec) *proxypkg.HealthPolicy {
	if h == nil {
		return nil
	}
	return &proxypkg.HealthPolicy{
		Path:               h.Path,
		IntervalMs:         h.IntervalMs,
		TimeoutMs:          h.TimeoutMs,
		HealthyThreshold:   h.HealthyThreshold,
		UnhealthyThreshold: h.UnhealthyThreshold,
	}
}

// warnOnLegacyRepo checks for the old /opt/conoha/<name>.git bare repo and
// returns a non-nil (non-fatal) error if present, so users can migrate cleanly.
func warnOnLegacyRepo(cli *ssh.Client, name string) error {
	cmdStr := fmt.Sprintf("test -d /opt/conoha/%s.git && echo yes || echo no", name)
	var buf bytes.Buffer
	_, err := internalssh.RunCommand(cli, cmdStr, &buf, os.Stderr)
	if err != nil {
		return nil
	}
	if strings.TrimSpace(buf.String()) == "yes" {
		return fmt.Errorf("legacy git bare repo /opt/conoha/%s.git exists (left untouched). Remove it after migration with 'rm -rf /opt/conoha/%s.git' via SSH", name, name)
	}
	return nil
}
