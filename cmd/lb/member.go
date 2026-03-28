package lb

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
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

	memberShowCmd := &cobra.Command{
		Use: "show <id>", Short: "Show member details",
		Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			poolID, _ := cmd.Flags().GetString("pool-id")
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			item, err := api.NewLoadBalancerAPI(client).GetMember(poolID, args[0])
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, item)
		},
	}
	memberShowCmd.Flags().String("pool-id", "", "pool ID (required)")
	_ = memberShowCmd.MarkFlagRequired("pool-id")
	memberCmd.AddCommand(memberShowCmd)

	memberCreateCmd := &cobra.Command{
		Use: "create", Short: "Create a pool member",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			address, _ := cmd.Flags().GetString("address")
			port, _ := cmd.Flags().GetInt("port")
			poolID, _ := cmd.Flags().GetString("pool-id")

			var weightPtr *int
			if cmd.Flags().Changed("weight") {
				w, _ := cmd.Flags().GetInt("weight")
				weightPtr = &w
			}

			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			lbAPI := api.NewLoadBalancerAPI(client)
			item, err := lbAPI.CreateMember(poolID, name, address, port, weightPtr)
			if err != nil {
				return err
			}
			if err := cmdutil.FormatOutput(cmd, item); err != nil {
				return err
			}
			if wc := cmdutil.GetWaitConfig(cmd, "member "+name); wc != nil {
				fmt.Fprintf(os.Stderr, "Waiting for member %s to become active...\n", name)
				return waitForLBResource(lbAPI, "member", item.ID, poolID, wc)
			}
			return nil
		},
	}
	memberCreateCmd.Flags().String("name", "", "member name (required)")
	memberCreateCmd.Flags().String("address", "", "IP address (required)")
	memberCreateCmd.Flags().Int("port", 0, "protocol port (required)")
	memberCreateCmd.Flags().String("pool-id", "", "pool ID (required)")
	memberCreateCmd.Flags().Int("weight", 1, "load balancing weight")
	_ = memberCreateCmd.MarkFlagRequired("name")
	_ = memberCreateCmd.MarkFlagRequired("address")
	_ = memberCreateCmd.MarkFlagRequired("port")
	_ = memberCreateCmd.MarkFlagRequired("pool-id")
	cmdutil.AddWaitFlags(memberCreateCmd)
	memberCmd.AddCommand(memberCreateCmd)

	memberDeleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a pool member",
		Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			poolID, _ := cmd.Flags().GetString("pool-id")
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			lbAPI := api.NewLoadBalancerAPI(client)
			item, err := lbAPI.GetMember(poolID, args[0])
			if err != nil {
				return err
			}
			ok, err := prompt.Confirm(fmt.Sprintf("Delete member %q (%s)?", item.Name, item.ID))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			if err := lbAPI.DeleteMember(poolID, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Member %s deleted\n", args[0])
			return nil
		},
	}
	memberDeleteCmd.Flags().String("pool-id", "", "pool ID (required)")
	_ = memberDeleteCmd.MarkFlagRequired("pool-id")
	memberCmd.AddCommand(memberDeleteCmd)
}
