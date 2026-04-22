package server

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
)

func init() {
	openPortCmd.Flags().String("sg", "", "security group name to add the rule to (default: <server-name>-sg; auto-created if missing)")
	openPortCmd.Flags().String("remote-ip", "0.0.0.0/0", "remote IP CIDR allowed by the rule")
	openPortCmd.Flags().String("protocol", "tcp", "IP protocol (tcp, udp)")
}

var openPortCmd = &cobra.Command{
	Use:   "open-port <server> <ports>",
	Short: "Open ingress ports via a custom security group",
	Long: `Open one or more ingress ports by adding rules to a custom security group
attached to the server. If the server has no custom security group, one
named "<server-name>-sg" is created and attached.

Port format: comma-separated list of single ports or ranges
    7860
    7860,8080
    7860,8080,9000-9010
`,
	Args: cmdutil.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)
		network := api.NewNetworkAPI(client)

		server, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		ranges, err := parsePortRanges(args[1])
		if err != nil {
			return err
		}

		sgNameOverride, _ := cmd.Flags().GetString("sg")
		remoteIP, _ := cmd.Flags().GetString("remote-ip")
		protocol, _ := cmd.Flags().GetString("protocol")
		switch protocol {
		case "tcp", "udp":
		default:
			return fmt.Errorf("unsupported --protocol %q (want tcp or udp)", protocol)
		}

		sgName := sgNameOverride
		if sgName == "" {
			sgName = server.Name + "-sg"
		}

		sg, err := ensureAttachedSG(network, server, sgName)
		if err != nil {
			return err
		}

		for _, r := range ranges {
			min := r.min
			max := r.max
			desc := strconv.Itoa(min)
			if min != max {
				desc = fmt.Sprintf("%d-%d", min, max)
			}
			if _, err := network.CreateSecurityGroupRule(sg.ID, "ingress", protocol, "IPv4", &min, &max, remoteIP); err != nil {
				return fmt.Errorf("adding rule for %s: %w", desc, err)
			}
			fmt.Fprintf(os.Stderr, "Added %s ingress rule %s from %s on security group %q\n", protocol, desc, remoteIP, sg.Name)
		}

		return nil
	},
}

// ensureAttachedSG returns the security group named sgName attached to every
// port of the server, creating the SG and/or attaching it as needed.
//
// The Server model does not carry a security-groups list; attachment lives on
// the server's Neutron ports. We inspect ports once and cross-reference their
// SG IDs against the tenant SG list.
func ensureAttachedSG(network *api.NetworkAPI, server *model.Server, sgName string) (*model.SecurityGroup, error) {
	sgs, err := network.ListSecurityGroups()
	if err != nil {
		return nil, err
	}
	var existing *model.SecurityGroup
	for i := range sgs {
		if sgs[i].Name == sgName {
			existing = &sgs[i]
			break
		}
	}

	if existing != nil {
		ports, err := network.ListPortsByDevice(server.ID)
		if err != nil {
			return nil, err
		}
		attached := len(ports) > 0
		for _, p := range ports {
			has := false
			for _, id := range p.SecurityGroups {
				if id == existing.ID {
					has = true
					break
				}
			}
			if !has {
				attached = false
				break
			}
		}
		if attached {
			return existing, nil
		}
	} else {
		fmt.Fprintf(os.Stderr, "==> Creating security group %q\n", sgName)
		created, err := network.CreateSecurityGroup(sgName, fmt.Sprintf("auto-created by 'conoha server open-port' for %s", server.Name))
		if err != nil {
			return nil, fmt.Errorf("creating security group %q: %w", sgName, err)
		}
		existing = created
	}

	fmt.Fprintf(os.Stderr, "==> Attaching security group %q to server %s\n", sgName, server.Name)
	if err := network.AddServerSecurityGroup(server.ID, sgName); err != nil {
		return nil, fmt.Errorf("attaching security group %q to server: %w", sgName, err)
	}
	return existing, nil
}

type portRange struct{ min, max int }

// parsePortRanges parses a comma-separated list of single ports or ranges
// ("7860", "7860,8080", "9000-9010") into a deduped, validated list.
func parsePortRanges(spec string) ([]portRange, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("no ports specified")
	}
	var out []portRange
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			lo, hi, ok := strings.Cut(part, "-")
			if !ok {
				return nil, fmt.Errorf("invalid port range %q", part)
			}
			lmin, err := parsePort(strings.TrimSpace(lo))
			if err != nil {
				return nil, err
			}
			lmax, err := parsePort(strings.TrimSpace(hi))
			if err != nil {
				return nil, err
			}
			if lmin > lmax {
				return nil, fmt.Errorf("invalid port range %q (min > max)", part)
			}
			out = append(out, portRange{min: lmin, max: lmax})
			continue
		}
		p, err := parsePort(part)
		if err != nil {
			return nil, err
		}
		out = append(out, portRange{min: p, max: p})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no ports specified")
	}
	return out, nil
}

func parsePort(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: not a number", s)
	}
	if n < 1 || n > 65535 {
		return 0, fmt.Errorf("invalid port %d (must be 1-65535)", n)
	}
	return n, nil
}
