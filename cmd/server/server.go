package server

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
)

// Cmd is the server command group.
var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Manage compute servers",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(renameCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(rebootCmd)
	Cmd.AddCommand(resizeCmd)
	Cmd.AddCommand(rebuildCmd)
	Cmd.AddCommand(consoleCmd)
	Cmd.AddCommand(ipsCmd)
	Cmd.AddCommand(metadataCmd)
	Cmd.AddCommand(attachVolumeCmd)
	Cmd.AddCommand(detachVolumeCmd)
	Cmd.AddCommand(sshCmd)
	Cmd.AddCommand(deployCmd)
	Cmd.AddCommand(addSecurityGroupCmd)
	Cmd.AddCommand(removeSecurityGroupCmd)
}

func getComputeAPI(cmd *cobra.Command) (*api.ComputeAPI, error) {
	client, err := cmdutil.NewClient(cmd)
	if err != nil {
		return nil, err
	}
	return api.NewComputeAPI(client), nil
}

// resolveServerID resolves an id-or-name argument to a server ID.
func resolveServerID(compute *api.ComputeAPI, idOrName string) (string, error) {
	s, err := compute.FindServer(idOrName)
	if err != nil {
		return "", err
	}
	return s.ID, nil
}

func formatMB(mb int) string {
	if mb == 0 {
		return "0"
	}
	if mb%1024 == 0 {
		return fmt.Sprintf("%dG", mb/1024)
	}
	return fmt.Sprintf("%dM", mb)
}
