package lb

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
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
		return cmdutil.FormatOutput(cmd, rows)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show load balancer details",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		lb, err := api.NewLoadBalancerAPI(client).GetLoadBalancer(args[0])
		if err != nil {
			return err
		}
		return cmdutil.FormatOutput(cmd, lb)
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
		return cmdutil.FormatOutput(cmd, lb)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a load balancer",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		lbAPI := api.NewLoadBalancerAPI(client)
		lb, err := lbAPI.GetLoadBalancer(args[0])
		if err != nil {
			return err
		}
		ok, err := prompt.Confirm(fmt.Sprintf("Delete load balancer %q (%s)?", lb.Name, lb.ID))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		if err := lbAPI.DeleteLoadBalancer(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Load balancer %s deleted\n", args[0])
		return nil
	},
}

// waitForLBResource polls a sub-resource until provisioning_status becomes ACTIVE or ERROR.
func waitForLBResource(lbAPI *api.LoadBalancerAPI, resourceType, id, poolID string, wc *cmdutil.WaitConfig) error {
	return cmdutil.WaitFor(*wc, func() (bool, string, error) {
		var status string
		switch resourceType {
		case "listener":
			r, err := lbAPI.GetListener(id)
			if err != nil {
				return false, "", err
			}
			status = r.ProvisioningStatus
		case "pool":
			r, err := lbAPI.GetPool(id)
			if err != nil {
				return false, "", err
			}
			status = r.ProvisioningStatus
		case "member":
			r, err := lbAPI.GetMember(poolID, id)
			if err != nil {
				return false, "", err
			}
			status = r.ProvisioningStatus
		case "healthmonitor":
			r, err := lbAPI.GetHealthMonitor(id)
			if err != nil {
				return false, "", err
			}
			status = r.ProvisioningStatus
		}
		if status == "ACTIVE" {
			return true, status, nil
		}
		if status == "ERROR" {
			return false, status, fmt.Errorf("%s %s entered ERROR state", resourceType, id)
		}
		return false, status, nil
	})
}
