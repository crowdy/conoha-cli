package lb

import (
	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
)

var memberCmd = &cobra.Command{Use: "member", Short: "Manage pool members"}

func init() {
	memberListCmd := &cobra.Command{
		Use: "list", Short: "List members",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			poolID, _ := cmd.Flags().GetString("pool-id")
			items, err := api.NewLoadBalancerAPI(client).ListMembers(poolID)
			if err != nil {
				return err
			}
			type row struct {
				ID              string `json:"id"`
				Name            string `json:"name"`
				Address         string `json:"address"`
				ProtocolPort    int    `json:"protocol_port"`
				Weight          int    `json:"weight"`
				OperatingStatus string `json:"operating_status"`
			}
			rows := make([]row, len(items))
			for i, m := range items {
				rows[i] = row{
					ID: m.ID, Name: m.Name, Address: m.Address,
					ProtocolPort: m.ProtocolPort, Weight: m.Weight,
					OperatingStatus: m.OperatingStatus,
				}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}
	memberListCmd.Flags().String("pool-id", "", "pool ID (required)")
	_ = memberListCmd.MarkFlagRequired("pool-id")
	memberCmd.AddCommand(memberListCmd)
}
