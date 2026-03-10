package server

import (
	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
)

func initClient(cmd *cobra.Command) (*api.Client, error) {
	return cmdutil.NewClient(cmd)
}
