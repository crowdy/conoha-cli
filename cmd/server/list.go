package server

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/output"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		servers, err := compute.ListServers()
		if err != nil {
			return err
		}

		flavors, err := compute.ListFlavors()
		if err != nil {
			return err
		}
		flavorMap := make(map[string]string, len(flavors))
		for _, f := range flavors {
			flavorMap[f.ID] = f.Name
		}

		type serverRow struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Flavor string `json:"flavor"`
			Tag    string `json:"tag"`
		}
		rows := make([]serverRow, len(servers))
		for i, s := range servers {
			flavorName := flavorMap[s.Flavor.ID]
			if flavorName == "" {
				flavorName = s.Flavor.ID
			}
			rows[i] = serverRow{
				ID:     s.ID,
				Name:   s.Name,
				Status: s.Status,
				Flavor: flavorName,
				Tag:    s.Metadata["instance_name_tag"],
			}
		}

		return cmdutil.FormatOutput(cmd, rows)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id|name>",
	Short: "Show server details",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)
		server, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		format := cmdutil.GetFormat(cmd)
		if format != "" && format != "table" {
			return output.New(format).Format(os.Stdout, server)
		}

		// Human-readable key-value output
		printServerDetail(server)

		// Volume attachments (non-fatal)
		if attachments, err := compute.ListVolumeAttachments(server.ID); err == nil && len(attachments) > 0 {
			volumeAPI := api.NewVolumeAPI(client)
			fmt.Println("Volumes:")
			for _, a := range attachments {
				size := ""
				if vol, err := volumeAPI.GetVolume(a.VolumeID); err == nil {
					size = fmt.Sprintf(" %dGB", vol.Size)
				}
				fmt.Printf("  %s %s%s\n", a.VolumeID, a.Device, size)
			}
		}

		// Ports (non-fatal)
		networkAPI := api.NewNetworkAPI(client)
		if ports, err := networkAPI.ListPortsByDevice(server.ID); err == nil && len(ports) > 0 {
			fmt.Println("Ports:")
			for _, p := range ports {
				var ips []string
				for _, ip := range p.FixedIPs {
					ips = append(ips, ip.IPAddress)
				}
				sgs := ""
				if len(p.SecurityGroups) > 0 {
					sgs = " sg=[" + strings.Join(p.SecurityGroups, ",") + "]"
				}
				fmt.Printf("  %s mac=%s ips=[%s]%s\n", p.ID, p.MACAddress, strings.Join(ips, ","), sgs)
			}
		}

		return nil
	},
}

func printServerDetail(s *model.Server) {
	jst := time.FixedZone("JST", 9*60*60)

	fmt.Printf("ID:        %s\n", s.ID)
	fmt.Printf("Name:      %s\n", s.Name)
	fmt.Printf("Status:    %s\n", s.Status)
	fmt.Printf("Flavor:    %s\n", s.Flavor.ID)
	fmt.Printf("Image:     %s\n", s.ImageID)
	fmt.Printf("Key Name:  %s\n", s.KeyName)
	fmt.Printf("Tenant:    %s\n", s.TenantID)
	fmt.Printf("Created:   %s (%s JST)\n",
		s.Created.Format(time.RFC3339),
		s.Created.In(jst).Format("2006-01-02 15:04"))
	if !s.Updated.IsZero() {
		fmt.Printf("Updated:   %s (%s JST)\n",
			s.Updated.Format(time.RFC3339),
			s.Updated.In(jst).Format("2006-01-02 15:04"))
	}

	if len(s.Addresses) > 0 {
		fmt.Println("Addresses:")
		for net, addrs := range s.Addresses {
			for _, a := range addrs {
				fmt.Printf("  %s: %s (v%d, %s)\n", net, a.Addr, a.Version, a.Type)
			}
		}
	}

	if len(s.Metadata) > 0 {
		fmt.Println("Metadata:")
		for k, v := range s.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}

var renameCmd = &cobra.Command{
	Use:   "rename <id|name> <new-name>",
	Short: "Rename a server",
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		server, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}
		renamed, err := compute.RenameServer(server.ID, args[1])
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server renamed: %s -> %s\n", args[0], renamed.Name)
		return nil
	},
}
