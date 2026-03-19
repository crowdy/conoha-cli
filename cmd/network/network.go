package network

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/prompt"
)

var Cmd = &cobra.Command{
	Use:   "network",
	Short: "Manage networks",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(subnetCmd)
	Cmd.AddCommand(portCmd)
	Cmd.AddCommand(sgCmd)
	Cmd.AddCommand(sgRuleCmd)
	Cmd.AddCommand(qosCmd)

	createCmd.Flags().String("name", "", "network name (required)")
	_ = createCmd.MarkFlagRequired("name")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List networks",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		networks, err := api.NewNetworkAPI(client).ListNetworks()
		if err != nil {
			return err
		}

		type row struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		rows := make([]row, len(networks))
		for i, n := range networks {
			rows[i] = row{ID: n.ID, Name: n.Name, Status: n.Status}
		}
		return cmdutil.FormatOutput(cmd, rows)
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a network",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		net, err := api.NewNetworkAPI(client).CreateNetwork(name)
		if err != nil {
			return err
		}
		return cmdutil.FormatOutput(cmd, net)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a network",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete network %s?", args[0]))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		if err := api.NewNetworkAPI(client).DeleteNetwork(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Network %s deleted\n", args[0])
		return nil
	},
}

// Subnet subcommands
var subnetCmd = &cobra.Command{
	Use:   "subnet",
	Short: "Manage subnets",
}

func init() {
	subnetListCmd := &cobra.Command{
		Use:   "list",
		Short: "List subnets",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			subnets, err := api.NewNetworkAPI(client).ListSubnets()
			if err != nil {
				return err
			}

			type row struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				NetworkID string `json:"network_id"`
				CIDR      string `json:"cidr"`
			}
			rows := make([]row, len(subnets))
			for i, s := range subnets {
				rows[i] = row{ID: s.ID, Name: s.Name, NetworkID: s.NetworkID, CIDR: s.CIDR}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}

	subnetCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a subnet",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			networkID, _ := cmd.Flags().GetString("network-id")
			cidr, _ := cmd.Flags().GetString("cidr")
			name, _ := cmd.Flags().GetString("name")
			ipVersion, _ := cmd.Flags().GetInt("ip-version")
			subnet, err := api.NewNetworkAPI(client).CreateSubnet(networkID, cidr, name, ipVersion)
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, subnet)
		},
	}
	subnetCreateCmd.Flags().String("network-id", "", "network ID (required)")
	subnetCreateCmd.Flags().String("cidr", "", "CIDR (required)")
	subnetCreateCmd.Flags().String("name", "", "subnet name")
	subnetCreateCmd.Flags().Int("ip-version", 4, "IP version (4 or 6)")
	_ = subnetCreateCmd.MarkFlagRequired("network-id")
	_ = subnetCreateCmd.MarkFlagRequired("cidr")

	subnetDeleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a subnet",
		Args:  cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete subnet %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewNetworkAPI(client).DeleteSubnet(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Subnet %s deleted\n", args[0])
			return nil
		},
	}

	subnetCmd.AddCommand(subnetListCmd)
	subnetCmd.AddCommand(subnetCreateCmd)
	subnetCmd.AddCommand(subnetDeleteCmd)
}

// Port subcommands
var portCmd = &cobra.Command{
	Use:   "port",
	Short: "Manage ports",
}

func init() {
	portListCmd := &cobra.Command{
		Use:   "list",
		Short: "List ports",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			ports, err := api.NewNetworkAPI(client).ListPorts()
			if err != nil {
				return err
			}

			type row struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				NetworkID string `json:"network_id"`
				Status    string `json:"status"`
			}
			rows := make([]row, len(ports))
			for i, p := range ports {
				rows[i] = row{ID: p.ID, Name: p.Name, NetworkID: p.NetworkID, Status: p.Status}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}

	portCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a port",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			networkID, _ := cmd.Flags().GetString("network-id")
			name, _ := cmd.Flags().GetString("name")
			port, err := api.NewNetworkAPI(client).CreatePort(networkID, name)
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, port)
		},
	}
	portCreateCmd.Flags().String("network-id", "", "network ID (required)")
	portCreateCmd.Flags().String("name", "", "port name")
	_ = portCreateCmd.MarkFlagRequired("network-id")

	portDeleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a port",
		Args:  cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete port %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewNetworkAPI(client).DeletePort(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Port %s deleted\n", args[0])
			return nil
		},
	}

	portCmd.AddCommand(portListCmd)
	portCmd.AddCommand(portCreateCmd)
	portCmd.AddCommand(portDeleteCmd)
}

// Security Group subcommands
var sgCmd = &cobra.Command{
	Use:     "security-group",
	Aliases: []string{"sg"},
	Short:   "Manage security groups",
}

func init() {
	sgListCmd := &cobra.Command{
		Use:   "list",
		Short: "List security groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			sgs, err := api.NewNetworkAPI(client).ListSecurityGroups()
			if err != nil {
				return err
			}

			type row struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			rows := make([]row, len(sgs))
			for i, sg := range sgs {
				rows[i] = row{ID: sg.ID, Name: sg.Name, Description: sg.Description}
			}
			return cmdutil.FormatOutput(cmd, rows)
		},
	}

	sgShowCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show security group details",
		Args:  cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			sg, err := api.NewNetworkAPI(client).GetSecurityGroup(args[0])
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, sg)
		},
	}

	sgCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a security group",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			desc, _ := cmd.Flags().GetString("description")
			sg, err := api.NewNetworkAPI(client).CreateSecurityGroup(name, desc)
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, sg)
		},
	}
	sgCreateCmd.Flags().String("name", "", "security group name (required)")
	sgCreateCmd.Flags().String("description", "", "description")
	_ = sgCreateCmd.MarkFlagRequired("name")

	sgDeleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a security group",
		Args:  cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete security group %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewNetworkAPI(client).DeleteSecurityGroup(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Security group %s deleted\n", args[0])
			return nil
		},
	}

	sgCmd.AddCommand(sgListCmd)
	sgCmd.AddCommand(sgShowCmd)
	sgCmd.AddCommand(sgCreateCmd)
	sgCmd.AddCommand(sgDeleteCmd)
}

// Security Group Rule subcommands
var sgRuleCmd = &cobra.Command{
	Use:     "security-group-rule",
	Aliases: []string{"sgr"},
	Short:   "Manage security group rules",
}

func init() {
	sgRuleListCmd := &cobra.Command{
		Use:   "list",
		Short: "List security group rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			rules, err := api.NewNetworkAPI(client).ListSecurityGroupRules()
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, rules)
		},
	}

	sgRuleCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a security group rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			sgID, _ := cmd.Flags().GetString("security-group-id")
			direction, _ := cmd.Flags().GetString("direction")
			protocol, _ := cmd.Flags().GetString("protocol")
			ethertype, _ := cmd.Flags().GetString("ethertype")
			portMin, _ := cmd.Flags().GetInt("port-min")
			portMax, _ := cmd.Flags().GetInt("port-max")
			remoteIP, _ := cmd.Flags().GetString("remote-ip")

			var pMin, pMax *int
			if cmd.Flags().Changed("port-min") {
				pMin = &portMin
			}
			if cmd.Flags().Changed("port-max") {
				pMax = &portMax
			}

			rule, err := api.NewNetworkAPI(client).CreateSecurityGroupRule(sgID, direction, protocol, ethertype, pMin, pMax, remoteIP)
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, rule)
		},
	}
	sgRuleCreateCmd.Flags().String("security-group-id", "", "security group ID (required)")
	sgRuleCreateCmd.Flags().String("direction", "ingress", "direction (ingress/egress)")
	sgRuleCreateCmd.Flags().String("protocol", "", "protocol (tcp/udp/icmp)")
	sgRuleCreateCmd.Flags().String("ethertype", "IPv4", "ethertype (IPv4/IPv6)")
	sgRuleCreateCmd.Flags().Int("port-min", 0, "minimum port")
	sgRuleCreateCmd.Flags().Int("port-max", 0, "maximum port")
	sgRuleCreateCmd.Flags().String("remote-ip", "", "remote IP prefix (CIDR)")
	_ = sgRuleCreateCmd.MarkFlagRequired("security-group-id")

	sgRuleDeleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a security group rule",
		Args:  cmdutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := prompt.Confirm(fmt.Sprintf("Delete security group rule %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := api.NewNetworkAPI(client).DeleteSecurityGroupRule(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Security group rule %s deleted\n", args[0])
			return nil
		},
	}

	sgRuleCmd.AddCommand(sgRuleListCmd)
	sgRuleCmd.AddCommand(sgRuleCreateCmd)
	sgRuleCmd.AddCommand(sgRuleDeleteCmd)
}

// QoS subcommand
var qosCmd = &cobra.Command{
	Use:   "qos",
	Short: "Manage QoS policies",
}

func init() {
	qosListCmd := &cobra.Command{
		Use:   "list",
		Short: "List QoS policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.NewClient(cmd)
			if err != nil {
				return err
			}
			policies, err := api.NewNetworkAPI(client).ListQoSPolicies()
			if err != nil {
				return err
			}
			return cmdutil.FormatOutput(cmd, policies)
		},
	}

	qosCmd.AddCommand(qosListCmd)
}
