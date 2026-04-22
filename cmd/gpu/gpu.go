// Package gpu provides opinionated GPU provisioning helpers — notably the
// NVIDIA driver + Container Toolkit install flow that every GPU-sample user
// repeats by hand after booting an L4 flavor.
package gpu

import (
	"github.com/spf13/cobra"
)

// Cmd is the gpu command group.
var Cmd = &cobra.Command{
	Use:   "gpu",
	Short: "GPU-specific provisioning shortcuts",
	Long:  "Operations that automate the post-boot steps required to turn a fresh GPU VPS into a Docker host that can schedule CUDA workloads.",
}

func init() {
	Cmd.AddCommand(setupCmd)
}
