package lb

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
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

	poolShowCmd := &cobra.Command{
		Use: "show <id>", Short: "Show pool details",
		Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			item, err := api.NewLoadBalancerAPI(client).GetPool(args[0])
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, item)
		},
	}
	poolCmd.AddCommand(poolShowCmd)

	poolCreateCmd := &cobra.Command{
		Use: "create", Short: "Create a pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			protocol, _ := cmd.Flags().GetString("protocol")
			lbAlgorithm, _ := cmd.Flags().GetString("lb-algorithm")
			listenerID, _ := cmd.Flags().GetString("listener-id")

			switch protocol {
			case "TCP", "UDP":
			default:
				return fmt.Errorf("invalid protocol %q: must be TCP or UDP", protocol)
			}
			switch lbAlgorithm {
			case "ROUND_ROBIN", "LEAST_CONNECTIONS":
			default:
				return fmt.Errorf("invalid lb-algorithm %q: must be ROUND_ROBIN or LEAST_CONNECTIONS", lbAlgorithm)
			}

			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			lbAPI := api.NewLoadBalancerAPI(client)
			item, err := lbAPI.CreatePool(name, protocol, lbAlgorithm, listenerID)
			if err != nil {
				return err
			}
			if err := cmdutil.FormatOutput(cmd, item); err != nil {
				return err
			}
			if wc := cmdutil.GetWaitConfig(cmd, "pool "+name); wc != nil {
				fmt.Fprintf(os.Stderr, "Waiting for pool %s to become active...\n", name)
				return waitForLBResource(lbAPI, "pool", item.ID, "", wc)
			}
			return nil
		},
	}
	poolCreateCmd.Flags().String("name", "", "pool name (required)")
	poolCreateCmd.Flags().String("protocol", "", "protocol: TCP, UDP (required)")
	poolCreateCmd.Flags().String("lb-algorithm", "", "algorithm: ROUND_ROBIN, LEAST_CONNECTIONS (required)")
	poolCreateCmd.Flags().String("listener-id", "", "listener ID (required)")
	_ = poolCreateCmd.MarkFlagRequired("name")
	_ = poolCreateCmd.MarkFlagRequired("protocol")
	_ = poolCreateCmd.MarkFlagRequired("lb-algorithm")
	_ = poolCreateCmd.MarkFlagRequired("listener-id")
	cmdutil.AddWaitFlags(poolCreateCmd)
	poolCmd.AddCommand(poolCreateCmd)

	poolDeleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a pool",
		Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			lbAPI := api.NewLoadBalancerAPI(client)
			item, err := lbAPI.GetPool(args[0])
			if err != nil {
				return err
			}
			ok, err := prompt.Confirm(fmt.Sprintf("Delete pool %q (%s)?", item.Name, item.ID))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			if err := lbAPI.DeletePool(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Pool %s deleted\n", args[0])
			return nil
		},
	}
	poolCmd.AddCommand(poolDeleteCmd)
}
