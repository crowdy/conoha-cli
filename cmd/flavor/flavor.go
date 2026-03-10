package flavor

import (
	"fmt"
	"os"
	"sort"

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

		sort.Slice(flavors, func(i, j int) bool {
			if flavors[i].VCPUs != flavors[j].VCPUs {
				return flavors[i].VCPUs < flavors[j].VCPUs
			}
			return flavors[i].RAM < flavors[j].RAM
		})

		type row struct {
			VCPUs int    `json:"vcpus"`
			RAM   string `json:"ram"`
			Disk  string `json:"disk"`
			ID    string `json:"id"`
			Name  string `json:"name"`
		}
		rows := make([]row, len(flavors))
		for i, f := range flavors {
			rows[i] = row{
				VCPUs: f.VCPUs,
				RAM:   formatMB(f.RAM),
				Disk:  formatDiskGB(f.Disk),
				ID:    f.ID,
				Name:  f.Name,
			}
		}

		format := cmdutil.GetFormat(cmd)
		if err := output.New(format).Format(os.Stdout, rows); err != nil {
			return err
		}
		if format == "" || format == "table" {
			fmt.Fprintln(os.Stderr, "\nNote: Some flavors may be restricted to prevent abuse. If you cannot use a flavor,")
			fmt.Fprintln(os.Stderr, "please contact ConoHa support: https://www.conoha.jp/conoha/contact/")
		}
		return nil
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

func formatMB(mb int) string {
	if mb == 0 {
		return "0"
	}
	if mb%1024 == 0 {
		return fmt.Sprintf("%dG", mb/1024)
	}
	return fmt.Sprintf("%dM", mb)
}

func formatDiskGB(gb int) string {
	if gb == 0 {
		return "0"
	}
	return fmt.Sprintf("%dG", gb)
}
