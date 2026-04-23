package server

import (
	"fmt"
	"net"
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
	openPortCmd.Flags().String("remote-ip", "0.0.0.0/0", "remote IP CIDR (IPv4 or IPv6) allowed by the rule")
	openPortCmd.Flags().String("protocol", "tcp", "IP protocol (tcp or udp; icmp not supported)")
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

		ethertype, err := ethertypeFromCIDR(remoteIP)
		if err != nil {
			return err
		}

		sgName := sgNameOverride
		if sgName == "" {
			sgName = server.Name + "-sg"
		}

		sg, err := ensureAttachedSG(network, server, sgName)
		if err != nil {
			return err
		}

		toCreate, skipped := filterExistingRanges(ranges, sg.Rules, protocol, remoteIP, ethertype)
		for _, r := range skipped {
			fmt.Fprintf(os.Stderr, "Skipped %s (rule already present on %q)\n", rangeDesc(r), sg.Name)
		}

		var added, failed int
		var firstErr error
		for _, r := range toCreate {
			min := r.min
			max := r.max
			desc := rangeDesc(r)
			if _, err := network.CreateSecurityGroupRule(sg.ID, "ingress", protocol, ethertype, &min, &max, remoteIP); err != nil {
				failed++
				if firstErr == nil {
					firstErr = fmt.Errorf("adding rule for %s: %w", desc, err)
				}
				fmt.Fprintf(os.Stderr, "Failed to add %s: %v\n", desc, err)
				continue
			}
			added++
			fmt.Fprintf(os.Stderr, "Added %s ingress rule %s from %s on security group %q\n", protocol, desc, remoteIP, sg.Name)
		}

		fmt.Fprintf(os.Stderr, "Summary: %d added, %d skipped, %d failed (of %d)\n", added, len(skipped), failed, len(ranges))
		return firstErr
	},
}

func rangeDesc(r portRange) string {
	if r.min == r.max {
		return strconv.Itoa(r.min)
	}
	return fmt.Sprintf("%d-%d", r.min, r.max)
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
	existing, dupes := pickSGByName(sgs, sgName)
	if len(dupes) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: multiple security groups named %q found; using %s (other IDs: %s)\n",
			sgName, existing.ID, strings.Join(dupes, ", "))
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
// First-occurrence order is preserved.
func parsePortRanges(spec string) ([]portRange, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("no ports specified")
	}
	seen := make(map[portRange]bool)
	var out []portRange
	add := func(pr portRange) {
		if seen[pr] {
			return
		}
		seen[pr] = true
		out = append(out, pr)
	}
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
			add(portRange{min: lmin, max: lmax})
			continue
		}
		p, err := parsePort(part)
		if err != nil {
			return nil, err
		}
		add(portRange{min: p, max: p})
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

// ethertypeFromCIDR returns "IPv4" or "IPv6" based on the parsed CIDR. The
// Neutron ethertype must match the remote_ip_prefix family or the rule is
// rejected with an opaque error; we validate and classify locally.
func ethertypeFromCIDR(cidr string) (string, error) {
	if cidr == "" {
		return "", fmt.Errorf("remote-ip CIDR is empty")
	}
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid remote-ip CIDR %q: %w", cidr, err)
	}
	if ip.To4() != nil {
		return "IPv4", nil
	}
	return "IPv6", nil
}

// filterExistingRanges splits input ranges into (new, skipped) by matching
// against existing ingress rules on the same SG. A rule matches when
// direction/protocol/ethertype/remote_ip_prefix and port_range_min/max all
// align with the desired tuple.
func filterExistingRanges(ranges []portRange, existing []model.SecurityGroupRule, protocol, remoteIP, ethertype string) (newRanges, skipped []portRange) {
	has := make(map[portRange]bool)
	for _, r := range existing {
		if r.Direction != "ingress" || r.Protocol != protocol || r.EtherType != ethertype || r.RemoteIPPrefix != remoteIP {
			continue
		}
		if r.PortRangeMin == nil || r.PortRangeMax == nil {
			continue
		}
		has[portRange{min: *r.PortRangeMin, max: *r.PortRangeMax}] = true
	}
	for _, pr := range ranges {
		if has[pr] {
			skipped = append(skipped, pr)
			continue
		}
		newRanges = append(newRanges, pr)
	}
	return
}

// pickSGByName returns the first SG matching name and the IDs of any
// additional duplicates (OpenStack permits same-named SGs within a tenant).
func pickSGByName(sgs []model.SecurityGroup, name string) (*model.SecurityGroup, []string) {
	var first *model.SecurityGroup
	var dupes []string
	for i := range sgs {
		if sgs[i].Name != name {
			continue
		}
		if first == nil {
			first = &sgs[i]
			continue
		}
		dupes = append(dupes, sgs[i].ID)
	}
	return first, dupes
}
