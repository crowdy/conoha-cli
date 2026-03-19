package server

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

func init() {
	rebootCmd.Flags().Bool("hard", false, "perform hard reboot")
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id|name>",
	Short: "Delete a server",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		s, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}
		ok, err := prompt.Confirm(fmt.Sprintf("Delete server %q (%s)? This cannot be undone", s.Name, s.ID))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		if err := compute.DeleteServer(s.ID); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Server %s deleted\n", s.Name)
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start <id|name>",
	Short: "Start a server",
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
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		ok, err := prompt.Confirm("Resize server? This may cause downtime")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
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
	Args:  cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		ok, err := prompt.Confirm("Rebuild server? All data will be lost")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
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
		resp, err := compute.GetConsole(id)
		if err != nil {
			return err
		}
		fmt.Println(resp.RemoteConsole.URL)
		return nil
	},
}
