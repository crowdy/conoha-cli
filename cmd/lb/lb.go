package lb

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/output"
)

var Cmd = &cobra.Command{
	Use:   "lb",
	Short: "Manage load balancers",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(listenerCmd)
	Cmd.AddCommand(poolCmd)
	Cmd.AddCommand(memberCmd)
	Cmd.AddCommand(healthMonitorCmd)

	createCmd.Flags().String("name", "", "load balancer name (required)")
	createCmd.Flags().String("subnet-id", "", "VIP subnet ID (required)")
	_ = createCmd.MarkFlagRequired("name")
	_ = createCmd.MarkFlagRequired("subnet-id")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List load balancers",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		lbs, err := api.NewLoadBalancerAPI(client).ListLoadBalancers()
		if err != nil {
			return err
		}

		type row struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"provisioning_status"`
			VIP    string `json:"vip_address"`
		}
		rows := make([]row, len(lbs))
		for i, l := range lbs {
			rows[i] = row{ID: l.ID, Name: l.Name, Status: l.ProvisioningStatus, VIP: l.VipAddress}
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, rows)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show load balancer details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		lb, err := api.NewLoadBalancerAPI(client).GetLoadBalancer(args[0])
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, lb)
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a load balancer",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		subnetID, _ := cmd.Flags().GetString("subnet-id")
		lb, err := api.NewLoadBalancerAPI(client).CreateLoadBalancer(name, subnetID)
		if err != nil {
			return err
		}
		return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, lb)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a load balancer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		if err := api.NewLoadBalancerAPI(client).DeleteLoadBalancer(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Load balancer %s deleted\n", args[0])
		return nil
	},
}

// Listener subcommands
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
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, items)
		},
	}
	listenerCmd.AddCommand(listenerListCmd)
}

// Pool subcommands
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
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, items)
		},
	}
	poolCmd.AddCommand(poolListCmd)
}

// Member subcommands
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
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, items)
		},
	}
	memberListCmd.Flags().String("pool-id", "", "pool ID (required)")
	_ = memberListCmd.MarkFlagRequired("pool-id")
	memberCmd.AddCommand(memberListCmd)
}

// Health Monitor subcommands
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
			return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, items)
		},
	}
	healthMonitorCmd.AddCommand(hmListCmd)
}
