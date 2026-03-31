package server

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	cerrors "github.com/crowdy/conoha-cli/internal/errors"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

func init() {
	rebootCmd.Flags().Bool("hard", false, "perform hard reboot")
	for _, c := range []*cobra.Command{deleteCmd, startCmd, stopCmd, rebootCmd} {
		cmdutil.AddWaitFlags(c)
	}
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

		if wc := cmdutil.GetWaitConfig(cmd, "server "+s.Name); wc != nil {
			fmt.Fprintf(os.Stderr, "Waiting for server %s to be removed...\n", s.Name)
			if err := cmdutil.WaitFor(cmdutil.WaitConfig{Resource: wc.Resource, Timeout: wc.Timeout}, func() (bool, string, error) {
				_, err := compute.GetServer(s.ID)
				if err != nil {
					var nfe *cerrors.NotFoundError
					if errors.As(err, &nfe) {
						return true, "", nil
					}
					return false, "", err
				}
				return false, "deleting", nil
			}); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Server %s removed.\n", s.Name)
		}

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

		if wc := cmdutil.GetWaitConfig(cmd, "server "+args[0]); wc != nil {
			if err := waitForServerStatus(compute, id, "ACTIVE", wc); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Server %s is active.\n", args[0])
		}

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

		if wc := cmdutil.GetWaitConfig(cmd, "server "+args[0]); wc != nil {
			if err := waitForServerStatus(compute, id, "SHUTOFF", wc); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Server %s is stopped.\n", args[0])
		}

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

		if wc := cmdutil.GetWaitConfig(cmd, "server "+args[0]); wc != nil {
			// Wait for server to enter REBOOT state first, then wait for ACTIVE.
			// Without this, the poll may see ACTIVE before the reboot starts.
			if err := waitForServerReboot(compute, id, wc); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Server %s is active.\n", args[0])
		}

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
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)
		id, err := resolveServerID(compute, args[0])
		if err != nil {
			return err
		}
		// Verify image exists before proceeding
		imageAPI := api.NewImageAPI(client)
		img, err := imageAPI.GetImage(args[1])
		if err != nil {
			return fmt.Errorf("image %s is not available — rebuild is not possible.\nTo rebuild with a different image, specify one explicitly: --image <image-id>", args[1])
		}
		ok, err := prompt.Confirm(fmt.Sprintf("Rebuild server with %s? All data will be lost", img.Name))
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
		fmt.Fprintf(os.Stderr, "Server %s rebuilding with image %s\n", args[0], img.Name)
		return nil
	},
}

// waitForServerReboot waits for the server to enter a reboot state, then waits for ACTIVE.
// This avoids the race where the server is still ACTIVE when polling starts.
func waitForServerReboot(compute *api.ComputeAPI, id string, wc *cmdutil.WaitConfig) error {
	// Phase 1: wait for server to leave ACTIVE (enter REBOOT/HARD_REBOOT)
	_ = cmdutil.WaitFor(cmdutil.WaitConfig{
		Resource: wc.Resource,
		Timeout:  30 * time.Second,
		Interval: 2 * time.Second,
	}, func() (bool, string, error) {
		s, err := compute.GetServer(id)
		if err != nil {
			return false, "", err
		}
		if s.Status != "ACTIVE" {
			return true, s.Status, nil
		}
		return false, s.Status, nil
	})
	// Phase 2: wait for ACTIVE
	return waitForServerStatus(compute, id, "ACTIVE", wc)
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
