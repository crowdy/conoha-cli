package app

import (
	"bytes"
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

		sshClient, s, ip, err := connectToServer(cmd, args[0])
		if err != nil {
			return err
		}
		defer func() { _ = sshClient.Close() }()

		dataDir, _ := cmd.Flags().GetString("data-dir")
		client := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))

		if err := warnOnLegacyRepo(sshClient, pf.Name); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "==> Registering service %q on %s (%s)\n", pf.Name, s.Name, ip)
		svc, err := client.Upsert(proxypkg.UpsertRequest{
			Name:         pf.Name,
			Hosts:        pf.Hosts,
			HealthPolicy: mapHealth(pf.Health),
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Service %q registered. phase=%s tls=%s\n", svc.Name, svc.Phase, svc.TLSStatus)
		fmt.Fprintf(os.Stderr, "Next: run 'conoha app deploy %s' to push your app.\n", args[0])
		return nil
	},
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
