package lb

import (
	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
)

var poolCmd = &cobra.Command{Use: "pool", Short: "Manage pools"}

func init() {
	poolListCmd := &cobra.Command{
		Use: "list", Short: "List pools",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			items, err := api.NewLoadBalancerAPI(client).ListPools()
			if err != nil {
				return err
			}
			type row struct {
				ID              string `json:"id"`
				Name            string `json:"name"`
				Protocol        string `json:"protocol"`
				LBMethod        string `json:"lb_algorithm"`
				OperatingStatus string `json:"operating_status"`
			}
			rows := make([]row, len(items))
			for i, p := range items {
				rows[i] = row{
					ID: p.ID, Name: p.Name, Protocol: p.Protocol,
					LBMethod: p.LBMethod, OperatingStatus: p.OperatingStatus,
				}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}
	poolCmd.AddCommand(poolListCmd)
}
