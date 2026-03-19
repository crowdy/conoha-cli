package server

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/output"
)

var ipsCmd = &cobra.Command{
	Use:   "ips <id|name>",
	Short: "Show server IP addresses",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		server, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		format := cmdutil.GetFormat(cmd)
		if format != "" && format != "table" {
			return output.New(format).Format(os.Stdout, server.Addresses)
		}

		for net, addrs := range server.Addresses {
			for _, a := range addrs {
				fmt.Printf("%s: %s (v%d, %s)\n", net, a.Addr, a.Version, a.Type)
			}
		}
		return nil
	},
}

var metadataCmd = &cobra.Command{
	Use:   "metadata <id|name>",
	Short: "Show server metadata",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		meta, err := compute.GetServerMetadata(id)
		if err != nil {
			return err
		}

		format := cmdutil.GetFormat(cmd)
		if format != "" && format != "table" {
			return output.New(format).Format(os.Stdout, meta)
		}

		for k, v := range meta {
			fmt.Printf("%s: %s\n", k, v)
		}
		return nil
	},
}
