package lb

import (
	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
)

var listenerCmd = &cobra.Command{Use: "listener", Short: "Manage listeners"}

func init() {
	listenerListCmd := &cobra.Command{
		Use: "list", Short: "List listeners",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			items, err := api.NewLoadBalancerAPI(client).ListListeners()
			if err != nil {
				return err
			}
			type row struct {
				ID              string `json:"id"`
				Name            string `json:"name"`
				Protocol        string `json:"protocol"`
				ProtocolPort    int    `json:"protocol_port"`
				OperatingStatus string `json:"operating_status"`
				LBID            string `json:"lb_id"`
			}
			rows := make([]row, len(items))
			for i, l := range items {
				lbID := ""
				if len(l.Loadbalancers) > 0 {
					lbID = l.Loadbalancers[0].ID
				}
				rows[i] = row{
					ID: l.ID, Name: l.Name, Protocol: l.Protocol,
					ProtocolPort: l.ProtocolPort, OperatingStatus: l.OperatingStatus,
					LBID: lbID,
				}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}
	listenerCmd.AddCommand(listenerListCmd)
}
