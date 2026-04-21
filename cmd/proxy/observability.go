package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

var logsCmd = &cobra.Command{
	Use:   "logs <server>",
	Short: "Show conoha-proxy container logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		container, _ := cmd.Flags().GetString("container")
		follow, _ := cmd.Flags().GetBool("follow")
		linesStr, _ := cmd.Flags().GetString("tail")
		lines, _ := strconv.Atoi(linesStr)
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		code, err := internalssh.RunScript(ctx.Client, proxypkg.LogsScript(container, follow, lines), nil, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("logs script exited with %d", code)
		}
		return nil
	},
}

var detailsCmd = &cobra.Command{
	Use:   "details <server>",
	Short: "Show conoha-proxy version and readiness",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		client := newAdminClient(ctx, dataDir)
		services, listErr := client.List()

		exec := &proxypkg.SSHExecutor{Client: ctx.Client}
		versionBody, vErr := curlVia(exec, dataDir, "/version")
		readyBody, rErr := curlVia(exec, dataDir, "/readyz")

		fmt.Printf("Server:  %s (%s)\n", ctx.Server.Name, ctx.IP)
		fmt.Printf("Version: %s\n", jsonField(versionBody, "version", vErr))
		fmt.Printf("Ready:   %s\n", jsonField(readyBody, "status", rErr))
		if listErr != nil {
			fmt.Printf("Services: (error: %v)\n", listErr)
		} else {
			fmt.Printf("Services: %d registered\n", len(services))
		}
		return nil
	},
}

var servicesCmd = &cobra.Command{
	Use:   "services <server>",
	Short: "List proxy services registered on the server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		client := newAdminClient(ctx, dataDir)
		services, err := client.List()
		if err != nil {
			return err
		}
		fmt.Printf("%-20s %-10s %-30s %s\n", "NAME", "PHASE", "ACTIVE", "HOSTS")
		for _, s := range services {
			active := "-"
			if s.ActiveTarget != nil {
				active = s.ActiveTarget.URL
			}
			hosts := ""
			for i, h := range s.Hosts {
				if i > 0 {
					hosts += ","
				}
				hosts += h
			}
			fmt.Printf("%-20s %-10s %-30s %s\n", s.Name, s.Phase, active, hosts)
		}
		return nil
	},
}

// curlVia is a tiny helper for non-/v1/ endpoints (/version, /readyz)
// that don't return the proxy error envelope.
func curlVia(exec proxypkg.Executor, dataDir, path string) ([]byte, error) {
	cmd := fmt.Sprintf("curl -sS --unix-socket %s/admin.sock http://admin%s", dataDir, path)
	var buf []byte
	w := byteWriter{&buf}
	if err := exec.Run(cmd, nil, &w); err != nil {
		return nil, err
	}
	return buf, nil
}

type byteWriter struct{ b *[]byte }

func (w *byteWriter) Write(p []byte) (int, error) { *w.b = append(*w.b, p...); return len(p), nil }

func jsonField(body []byte, key string, err error) string {
	if err != nil {
		return fmt.Sprintf("(error: %v)", err)
	}
	var m map[string]interface{}
	if uErr := json.Unmarshal(body, &m); uErr != nil {
		return fmt.Sprintf("(decode error: %v)", uErr)
	}
	v, ok := m[key]
	if !ok {
		return "(missing)"
	}
	return fmt.Sprint(v)
}

func init() {
	addSSHFlags(logsCmd)
	logsCmd.Flags().String("container", DefaultContainer, "docker container name")
	logsCmd.Flags().BoolP("follow", "f", false, "follow log output")
	logsCmd.Flags().String("tail", "0", "number of lines to show (0 = all)")

	for _, c := range []*cobra.Command{detailsCmd, servicesCmd} {
		addSSHFlags(c)
		c.Flags().String("data-dir", DefaultDataDir, "host data directory")
	}

	Cmd.AddCommand(logsCmd, detailsCmd, servicesCmd)
}
