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

		// Resolve flavor name
		flavorDisplay := server.Flavor.ID
		if f, err := compute.GetFlavor(server.Flavor.ID); err == nil {
			flavorDisplay = fmt.Sprintf("%s (%d vCPU, %s RAM)", f.Name, f.VCPUs, formatMB(f.RAM))
		}

		// Resolve image name
		imageDisplay := "(not set — booted from volume)"
		if server.ImageID != "" {
			imageAPI := api.NewImageAPI(client)
			if img, err := imageAPI.GetImage(server.ImageID); err == nil {
				imageDisplay = img.Name
			} else {
				imageDisplay = fmt.Sprintf("(deleted or unavailable) %s", server.ImageID)
			}
		}

		// Human-readable key-value output
		printServerDetail(server, flavorDisplay, imageDisplay)

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

		// Ports and Security Groups (non-fatal)
		networkAPI := api.NewNetworkAPI(client)
		if ports, err := networkAPI.ListPortsByDevice(server.ID); err == nil && len(ports) > 0 {
			// Build SG ID-to-name map
			sgMap := make(map[string]string)
			if sgs, err := networkAPI.ListSecurityGroups(); err == nil {
				for _, sg := range sgs {
					sgMap[sg.ID] = sg.Name
				}
			}

			fmt.Println("Ports:")
			for _, p := range ports {
				var ips []string
				for _, ip := range p.FixedIPs {
					ips = append(ips, ip.IPAddress)
				}
				fmt.Printf("  %s mac=%s ips=[%s]\n", p.ID, p.MACAddress, strings.Join(ips, ","))
			}

			// Collect unique SGs across all ports
			sgSeen := make(map[string]bool)
			var sgNames []string
			for _, p := range ports {
				for _, sgID := range p.SecurityGroups {
					if !sgSeen[sgID] {
						sgSeen[sgID] = true
						name := sgMap[sgID]
						if name == "" {
							name = sgID
						}
						sgNames = append(sgNames, name)
					}
				}
			}
			if len(sgNames) > 0 {
				fmt.Println("Security Groups:")
				for _, name := range sgNames {
					fmt.Printf("  %s\n", name)
				}
			}
		}

		return nil
	},
}

func printServerDetail(s *model.Server, flavorDisplay, imageDisplay string) {
	jst := time.FixedZone("JST", 9*60*60)

	fmt.Printf("ID:        %s\n", s.ID)
	fmt.Printf("Name:      %s\n", s.Name)
	fmt.Printf("Status:    %s\n", s.Status)
	fmt.Printf("Flavor:    %s\n", flavorDisplay)
	fmt.Printf("Image:     %s\n", imageDisplay)
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
	Short: "Rename a server (updates instance_name_tag)",
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
		_, err = compute.UpdateServerMetadata(server.ID, map[string]string{
			"instance_name_tag": args[1],
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server renamed: %s -> %s\n", args[0], args[1])
		return nil
	},
}
