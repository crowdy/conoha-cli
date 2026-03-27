package lb

import (
	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
)

var healthMonitorCmd = &cobra.Command{Use: "healthmonitor", Short: "Manage health monitors"}

func init() {
	hmListCmd := &cobra.Command{
		Use: "list", Short: "List health monitors",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			items, err := api.NewLoadBalancerAPI(client).ListHealthMonitors()
			if err != nil {
				return err
			}
			type row struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Type    string `json:"type"`
				Delay   int    `json:"delay"`
				Timeout int    `json:"timeout"`
				PoolID  string `json:"pool_id"`
			}
			rows := make([]row, len(items))
			for i, h := range items {
				poolID := ""
				if len(h.Pools) > 0 {
					poolID = h.Pools[0].ID
				}
				rows[i] = row{
					ID: h.ID, Name: h.Name, Type: h.Type,
					Delay: h.Delay, Timeout: h.Timeout, PoolID: poolID,
				}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}
	healthMonitorCmd.AddCommand(hmListCmd)
}
