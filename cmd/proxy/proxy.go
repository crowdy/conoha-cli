// Package proxy implements the `conoha proxy` command group for managing
// the conoha-proxy container on a ConoHa VPS.
package proxy

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// Cmd is the `conoha proxy` command group.
var Cmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage conoha-proxy reverse proxy on a ConoHa VPS",
}

// Defaults shared across subcommands.
const (
	DefaultImage     = "ghcr.io/crowdy/conoha-proxy:latest"
	DefaultDataDir   = "/var/lib/conoha-proxy"
	DefaultContainer = "conoha-proxy"
)

// SocketPath derives the admin socket path from the data directory.
func SocketPath(dataDir string) string {
	return dataDir + "/admin.sock"
}

// proxyContext bundles a live SSH connection with identifying metadata.
type proxyContext struct {
	Client *ssh.Client
	Server *model.Server
	IP     string
	User   string
}

// connect opens an SSH connection to the server named in args[0].
// Caller must Close() the returned client.
func connect(cmd *cobra.Command, args []string) (*proxyContext, error) {
	client, err := cmdutil.NewClient(cmd)
	if err != nil {
		return nil, err
	}
	compute := api.NewComputeAPI(client)
	s, err := compute.FindServer(args[0])
	if err != nil {
		return nil, err
	}
	ip, err := internalssh.ServerIP(s)
	if err != nil {
		return nil, err
	}

	user, _ := cmd.Flags().GetString("user")
	port, _ := cmd.Flags().GetString("port")
	identity, _ := cmd.Flags().GetString("identity")
	if identity == "" {
		identity = internalssh.ResolveKeyPath(s.KeyName)
	}
	if identity == "" {
		return nil, fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
	}
	sshClient, err := internalssh.Connect(internalssh.ConnectConfig{
		Host: ip, Port: port, User: user, KeyPath: identity,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}
	return &proxyContext{Client: sshClient, Server: s, IP: ip, User: user}, nil
}

// addSSHFlags registers the connection flags common to every proxy subcommand.
func addSSHFlags(c *cobra.Command) {
	c.Flags().StringP("user", "l", "root", "SSH user")
	c.Flags().StringP("port", "p", "22", "SSH port")
	c.Flags().StringP("identity", "i", "", "SSH private key path")
}

// newAdminClient returns a proxy Admin API client wired to the SSH connection.
func newAdminClient(ctx *proxyContext, dataDir string) *proxypkg.Client {
	exec := &proxypkg.SSHExecutor{Client: ctx.Client}
	return proxypkg.NewClient(exec, SocketPath(dataDir))
}
