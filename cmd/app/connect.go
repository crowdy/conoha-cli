package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/prompt"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

type appContext struct {
	Client      *ssh.Client
	AppName     string
	ServerID    string // the raw id-or-name value the user passed on the CLI
	Server      *model.Server
	IP          string
	User        string
	ComposeFile string
}

func connectToApp(cmd *cobra.Command, args []string) (*appContext, error) {
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

	appName, _ := cmd.Flags().GetString("app-name")
	if appName == "" {
		appName, err = prompt.String("App name")
		if err != nil {
			return nil, err
		}
	}
	// Legacy-tolerant: connectToApp is reached by logs/stop/restart/status/
	// rollback/destroy — all read/ops on already-deployed apps. Accepting the
	// pre-DNS-1123 form here lets users manage and tear down v0.1.x-era
	// deployments whose names contain uppercase or underscores. New deploys
	// go through the strict ValidateAppName on the init/deploy paths.
	if err := internalssh.ValidateAppNameExisting(appName); err != nil {
		return nil, err
	}

	user, _ := cmd.Flags().GetString("user")
	port, _ := cmd.Flags().GetString("port")
	identity, _ := cmd.Flags().GetString("identity")
	var composeFile string
	if f := cmd.Flags().Lookup("compose-file"); f != nil {
		composeFile = f.Value.String()
	}

	if identity == "" {
		identity = internalssh.ResolveKeyPath(s.KeyName)
	}
	if identity == "" {
		return nil, fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
	}

	sshClient, err := internalssh.Connect(internalssh.ConnectConfig{
		Host:    ip,
		Port:    port,
		User:    user,
		KeyPath: identity,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}

	return &appContext{
		Client:      sshClient,
		AppName:     appName,
		ServerID:    args[0],
		Server:      s,
		IP:          ip,
		User:        user,
		ComposeFile: composeFile,
	}, nil
}

func addAppFlags(cmd *cobra.Command) {
	cmd.Flags().String("app-name", "", "application name")
	cmd.Flags().StringP("user", "l", "root", "SSH user")
	cmd.Flags().StringP("port", "p", "22", "SSH port")
	cmd.Flags().StringP("identity", "i", "", "SSH private key path")
}
