package app

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/api"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	listCmd.Flags().StringP("user", "l", "root", "SSH user")
	listCmd.Flags().StringP("port", "p", "22", "SSH port")
	listCmd.Flags().StringP("identity", "i", "", "SSH private key path")
	listCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
}

var listCmd = &cobra.Command{
	Use:   "list <id|name>",
	Short: "List apps registered with conoha-proxy on a server",
	Long: `Enumerate apps by asking conoha-proxy for its registered services.

Columns: NAME, PHASE, ACTIVE, HOSTS. Empty output (exit 0) means the
proxy has no services registered — either 'conoha app init' hasn't
been run, or every app has been destroyed.

Legacy /opt/conoha/*.git scanning was removed. Apps deployed via
v0.1.x that were never migrated to the proxy model are not listed
here — use 'ssh <server> ls /opt/conoha/' manually.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(apiClient)

		s, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		ip, err := internalssh.ServerIP(s)
		if err != nil {
			return err
		}

		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetString("port")
		identity, _ := cmd.Flags().GetString("identity")
		dataDir, _ := cmd.Flags().GetString("data-dir")

		if identity == "" {
			identity = internalssh.ResolveKeyPath(s.KeyName)
		}
		if identity == "" {
			return fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
		}

		sshClient, err := internalssh.Connect(internalssh.ConnectConfig{
			Host:    ip,
			Port:    port,
			User:    user,
			KeyPath: identity,
		})
		if err != nil {
			return fmt.Errorf("SSH connect: %w", err)
		}
		defer func() { _ = sshClient.Close() }()

		admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))
		services, err := admin.List()
		if err != nil {
			return fmt.Errorf("list services: %w", err)
		}

		return printAppList(os.Stdout, services)
	},
}

// printAppList renders services as a padded NAME / PHASE / ACTIVE / HOSTS
// table. Empty input prints nothing (header with zero rows would look like
// corrupted state to scripts grepping by service name).
func printAppList(w io.Writer, services []proxypkg.Service) error {
	if len(services) == 0 {
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tPHASE\tACTIVE\tHOSTS")
	for _, svc := range services {
		active := "-"
		if svc.ActiveTarget != nil {
			active = svc.ActiveTarget.URL
		}
		hosts := strings.Join(svc.Hosts, ",")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", svc.Name, svc.Phase, active, hosts)
	}
	return tw.Flush()
}
