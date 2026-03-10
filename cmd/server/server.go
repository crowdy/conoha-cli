package server

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	"github.com/crowdy/conoha-cli/internal/output"
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

	createCmd.Flags().String("name", "", "server name (required)")
	createCmd.Flags().String("flavor", "", "flavor ID (required)")
	createCmd.Flags().String("image", "", "image ID")
	createCmd.Flags().String("key-name", "", "SSH key name")
	createCmd.Flags().String("admin-pass", "", "admin password")
	_ = createCmd.MarkFlagRequired("name")
	_ = createCmd.MarkFlagRequired("flavor")

	rebootCmd.Flags().Bool("hard", false, "perform hard reboot")
}

func getComputeAPI(cmd *cobra.Command) (*api.ComputeAPI, error) {
	client, err := cmdutil.NewClient(cmd)
	if err != nil {
		return nil, err
	}
	return api.NewComputeAPI(client), nil
}

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

		type serverRow struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		rows := make([]serverRow, len(servers))
		for i, s := range servers {
			rows[i] = serverRow{ID: s.ID, Name: s.Name, Status: s.Status}
		}

		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id|name>",
	Short: "Show server details",
	Args:  cobra.ExactArgs(1),
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
			return output.New(format).Format(os.Stdout, server)
		}

		// Human-readable key-value output
		printServerDetail(server)
		return nil
	},
}

func printServerDetail(s *model.Server) {
	jst := time.FixedZone("JST", 9*60*60)

	fmt.Printf("ID:        %s\n", s.ID)
	fmt.Printf("Name:      %s\n", s.Name)
	fmt.Printf("Status:    %s\n", s.Status)
	fmt.Printf("Flavor:    %s\n", s.FlavorID)
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
	Args:  cobra.ExactArgs(2),
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

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new server",
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		flavorID, _ := cmd.Flags().GetString("flavor")
		imageID, _ := cmd.Flags().GetString("image")
		keyName, _ := cmd.Flags().GetString("key-name")
		adminPass, _ := cmd.Flags().GetString("admin-pass")

		req := &model.ServerCreateRequest{}
		req.Server.Name = name
		req.Server.FlavorRef = flavorID
		req.Server.ImageRef = imageID
		req.Server.KeyName = keyName
		req.Server.AdminPass = adminPass

		server, err := compute.CreateServer(req)
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, server)
	},
}

// resolveServerID resolves an id-or-name argument to a server ID.
func resolveServerID(compute *api.ComputeAPI, idOrName string) (string, error) {
	s, err := compute.FindServer(idOrName)
	if err != nil {
		return "", err
	}
	return s.ID, nil
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id|name>",
	Short: "Delete a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.DeleteServer(id); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s deleted\n", args[0])
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start <id|name>",
	Short: "Start a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.StartServer(id); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s starting\n", args[0])
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <id|name>",
	Short: "Stop a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.StopServer(id); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s stopping\n", args[0])
		return nil
	},
}

var rebootCmd = &cobra.Command{
	Use:   "reboot <id|name>",
	Short: "Reboot a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		hard, _ := cmd.Flags().GetBool("hard")
		if err := compute.RebootServer(id, hard); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s rebooting\n", args[0])
		return nil
	},
}

var resizeCmd = &cobra.Command{
	Use:   "resize <id|name> <flavor-id>",
	Short: "Resize a server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.ResizeServer(id, args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s resizing to flavor %s\n", args[0], args[1])
		return nil
	},
}

var rebuildCmd = &cobra.Command{
	Use:   "rebuild <id|name> <image-id>",
	Short: "Rebuild a server with a new image",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		if err := compute.RebuildServer(id, args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s rebuilding with image %s\n", args[0], args[1])
		return nil
	},
}

var consoleCmd = &cobra.Command{
	Use:   "console <id|name>",
	Short: "Get VNC console URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		resp, err := compute.GetConsole(id)
		if err != nil {
			return err
		}
		fmt.Println(resp.Console.URL)
		return nil
	},
}

var ipsCmd = &cobra.Command{
	Use:   "ips <id|name>",
	Short: "Show server IP addresses",
	Args:  cobra.ExactArgs(1),
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
	Args:  cobra.ExactArgs(1),
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

var attachVolumeCmd = &cobra.Command{
	Use:   "attach-volume <server-id> <volume-id>",
	Short: "Attach a volume to a server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.AttachVolume(args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Volume %s attached to server %s\n", args[1], args[0])
		return nil
	},
}

var detachVolumeCmd = &cobra.Command{
	Use:   "detach-volume <server-id> <volume-id>",
	Short: "Detach a volume from a server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.DetachVolume(args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Volume %s detached from server %s\n", args[1], args[0])
		return nil
	},
}
