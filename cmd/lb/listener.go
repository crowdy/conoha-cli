package lb

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
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

	listenerShowCmd := &cobra.Command{
		Use: "show <id>", Short: "Show listener details",
		Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			item, err := api.NewLoadBalancerAPI(client).GetListener(args[0])
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, item)
		},
	}
	listenerCmd.AddCommand(listenerShowCmd)

	listenerCreateCmd := &cobra.Command{
		Use: "create", Short: "Create a listener",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			protocol, _ := cmd.Flags().GetString("protocol")
			port, _ := cmd.Flags().GetInt("port")
			lbID, _ := cmd.Flags().GetString("lb-id")

			switch protocol {
			case "TCP", "UDP":
			default:
				return fmt.Errorf("invalid protocol %q: must be TCP or UDP", protocol)
			}

			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			lbAPI := api.NewLoadBalancerAPI(client)
			item, err := lbAPI.CreateListener(name, protocol, port, lbID)
			if err != nil {
				return err
			}
			if err := cmdutil.FormatOutput(cmd, item); err != nil {
				return err
			}
			if wc := cmdutil.GetWaitConfig(cmd, "listener "+name); wc != nil {
				fmt.Fprintf(os.Stderr, "Waiting for listener %s to become active...\n", name)
				return waitForLBResource(lbAPI, "listener", item.ID, "", wc)
			}
			return nil
		},
	}
	listenerCreateCmd.Flags().String("name", "", "listener name (required)")
	listenerCreateCmd.Flags().String("protocol", "", "protocol: TCP, UDP (required)")
	listenerCreateCmd.Flags().Int("port", 0, "protocol port (required)")
	listenerCreateCmd.Flags().String("lb-id", "", "load balancer ID (required)")
	_ = listenerCreateCmd.MarkFlagRequired("name")
	_ = listenerCreateCmd.MarkFlagRequired("protocol")
	_ = listenerCreateCmd.MarkFlagRequired("port")
	_ = listenerCreateCmd.MarkFlagRequired("lb-id")
	cmdutil.AddWaitFlags(listenerCreateCmd)
	listenerCmd.AddCommand(listenerCreateCmd)

	listenerDeleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a listener",
		Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			lbAPI := api.NewLoadBalancerAPI(client)
			item, err := lbAPI.GetListener(args[0])
			if err != nil {
				return err
			}
			ok, err := prompt.Confirm(fmt.Sprintf("Delete listener %q (%s)?", item.Name, item.ID))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			if err := lbAPI.DeleteListener(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Listener %s deleted\n", args[0])
			return nil
		},
	}
	listenerCmd.AddCommand(listenerDeleteCmd)
}
