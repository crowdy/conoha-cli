package lb

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
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

	hmShowCmd := &cobra.Command{
		Use: "show <id>", Short: "Show health monitor details",
		Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			item, err := api.NewLoadBalancerAPI(client).GetHealthMonitor(args[0])
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, item)
		},
	}
	healthMonitorCmd.AddCommand(hmShowCmd)

	hmCreateCmd := &cobra.Command{
		Use: "create", Short: "Create a health monitor",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			poolID, _ := cmd.Flags().GetString("pool-id")
			monitorType, _ := cmd.Flags().GetString("type")
			delay, _ := cmd.Flags().GetInt("delay")
			timeout, _ := cmd.Flags().GetInt("timeout")
			maxRetries, _ := cmd.Flags().GetInt("max-retries")
			urlPath, _ := cmd.Flags().GetString("url-path")
			expectedCodes, _ := cmd.Flags().GetString("expected-codes")

			switch monitorType {
			case "TCP", "HTTP", "HTTPS", "PING", "UDP-CONNECT":
			default:
				return fmt.Errorf("invalid type %q: must be TCP, HTTP, HTTPS, PING, or UDP-CONNECT", monitorType)
			}
			if timeout >= delay {
				return fmt.Errorf("timeout (%d) must be less than delay (%d)", timeout, delay)
			}

			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			lbAPI := api.NewLoadBalancerAPI(client)
			item, err := lbAPI.CreateHealthMonitor(poolID, name, monitorType, delay, timeout, maxRetries, urlPath, expectedCodes)
			if err != nil {
				return err
			}
			if err := cmdutil.FormatOutput(cmd, item); err != nil {
				return err
			}
			if wc := cmdutil.GetWaitConfig(cmd, "healthmonitor "+name); wc != nil {
				fmt.Fprintf(os.Stderr, "Waiting for health monitor %s to become active...\n", name)
				return waitForLBResource(lbAPI, "healthmonitor", item.ID, "", wc)
			}
			return nil
		},
	}
	hmCreateCmd.Flags().String("name", "", "health monitor name (required)")
	hmCreateCmd.Flags().String("pool-id", "", "pool ID (required)")
	hmCreateCmd.Flags().String("type", "", "check type: TCP, HTTP, HTTPS, PING, UDP-CONNECT (required)")
	hmCreateCmd.Flags().Int("delay", 0, "interval in seconds between checks (required)")
	hmCreateCmd.Flags().Int("timeout", 0, "timeout in seconds for a check (required)")
	hmCreateCmd.Flags().Int("max-retries", 0, "failures before marking unhealthy (required)")
	hmCreateCmd.Flags().String("url-path", "", "URL path for HTTP checks")
	hmCreateCmd.Flags().String("expected-codes", "", "expected HTTP codes for healthy member")
	_ = hmCreateCmd.MarkFlagRequired("name")
	_ = hmCreateCmd.MarkFlagRequired("pool-id")
	_ = hmCreateCmd.MarkFlagRequired("type")
	_ = hmCreateCmd.MarkFlagRequired("delay")
	_ = hmCreateCmd.MarkFlagRequired("timeout")
	_ = hmCreateCmd.MarkFlagRequired("max-retries")
	cmdutil.AddWaitFlags(hmCreateCmd)
	healthMonitorCmd.AddCommand(hmCreateCmd)

	hmDeleteCmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a health monitor",
		Args: cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			lbAPI := api.NewLoadBalancerAPI(client)
			item, err := lbAPI.GetHealthMonitor(args[0])
			if err != nil {
				return err
			}
			ok, err := prompt.Confirm(fmt.Sprintf("Delete health monitor %q (%s)?", item.Name, item.ID))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			if err := lbAPI.DeleteHealthMonitor(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Health monitor %s deleted\n", args[0])
			return nil
		},
	}
	healthMonitorCmd.AddCommand(hmDeleteCmd)
}
