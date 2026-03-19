package server

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
)

var attachVolumeCmd = &cobra.Command{
	Use:   "attach-volume <server-id> <volume-id>",
	Short: "Attach a volume to a server",
	Args:  cmdutil.ExactArgs(2),
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
	Args:  cmdutil.ExactArgs(2),
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
