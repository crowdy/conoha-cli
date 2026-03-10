package server

import (
	"fmt"
	"os"

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
	Use:   "show <id>",
	Short: "Show server details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		server, err := compute.GetServer(args[0])
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, server)
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

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.DeleteServer(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s deleted\n", args[0])
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.StartServer(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s starting\n", args[0])
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <id>",
	Short: "Stop a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.StopServer(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s stopping\n", args[0])
		return nil
	},
}

var rebootCmd = &cobra.Command{
	Use:   "reboot <id>",
	Short: "Reboot a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		hard, _ := cmd.Flags().GetBool("hard")
		if err := compute.RebootServer(args[0], hard); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s rebooting\n", args[0])
		return nil
	},
}

var resizeCmd = &cobra.Command{
	Use:   "resize <id> <flavor-id>",
	Short: "Resize a server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.ResizeServer(args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s resizing to flavor %s\n", args[0], args[1])
		return nil
	},
}

var rebuildCmd = &cobra.Command{
	Use:   "rebuild <id> <image-id>",
	Short: "Rebuild a server with a new image",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		if err := compute.RebuildServer(args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s rebuilding with image %s\n", args[0], args[1])
		return nil
	},
}

var consoleCmd = &cobra.Command{
	Use:   "console <id>",
	Short: "Get VNC console URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		resp, err := compute.GetConsole(args[0])
		if err != nil {
			return err
		}
		fmt.Println(resp.Console.URL)
		return nil
	},
}

var ipsCmd = &cobra.Command{
	Use:   "ips <id>",
	Short: "Show server IP addresses",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		server, err := compute.GetServer(args[0])
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, server.Addresses)
	},
}

var metadataCmd = &cobra.Command{
	Use:   "metadata <id>",
	Short: "Show server metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		meta, err := compute.GetServerMetadata(args[0])
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, meta)
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
