package flavor

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/output"
)

var Cmd = &cobra.Command{
	Use:   "flavor",
	Short: "Manage server flavors",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available flavors",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)
		flavors, err := compute.ListFlavors()
		if err != nil {
			return err
		}

		type row struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			VCPUs int    `json:"vcpus"`
			RAM   int    `json:"ram"`
			Disk  int    `json:"disk"`
		}
		rows := make([]row, len(flavors))
		for i, f := range flavors {
			rows[i] = row{ID: f.ID, Name: f.Name, VCPUs: f.VCPUs, RAM: f.RAM, Disk: f.Disk}
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show flavor details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)
		flavor, err := compute.GetFlavor(args[0])
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, flavor)
	},
}
