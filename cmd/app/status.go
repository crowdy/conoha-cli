package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(statusCmd)
	statusCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	AddModeFlags(statusCmd)
}

// statusClient is the minimal proxy surface used by status. Kept as an
// interface so collectAppStatus is unit-testable without SSH.
type statusClient interface {
	Get(name string) (*proxypkg.Service, error)
}

// appStatusReport is the structured result of a `conoha app status` proxy
// query: the root service plus one entry per expose block. Shape is the
// public contract of `--format json`.
type appStatusReport struct {
	Root   *proxypkg.Service   `json:"root"`
	Expose []exposeStatusEntry `json:"expose"`
}

// exposeStatusEntry pairs an expose block's label with the proxy service it
// resolves to. Service may be nil if the proxy returned an error for this
// entry; the row is still emitted so consumers can see the gap.
type exposeStatusEntry struct {
	Label   string            `json:"label"`
	Service *proxypkg.Service `json:"service"`
}

var statusCmd = &cobra.Command{
	Use:   "status <id|name>",
	Short: "Show app container status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		mode, err := ResolveModeFromCtx(cmd, ctx)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return notInitializedError(ctx.AppName, ctx.ServerID, "")
			}
			return err
		}

		format := cmdutil.GetFormat(cmd)

		// In JSON mode we skip compose ps so stdout stays parseable as a
		// single JSON document. Table mode keeps the existing compose ps
		// dump on stdout so scripts that grep container state don't break.
		if format != "json" {
			var psCmd string
			if mode == ModeProxy {
				psCmd = buildStatusCmdForProxy(ctx.AppName)
			} else {
				psCmd = buildStatusCmdForNoProxy(ctx.AppName)
			}
			// A non-nil err from RunCommand is an SSH-layer failure (auth, transport,
			// channel close) — not something `compose ps` could produce by having no
			// containers to show. Surface it as a non-zero exit so scripts that check
			// $? don't silently get wrong information. Remote non-zero exit codes are
			// downgraded to a warning because an empty project is a normal state.
			code, err := internalssh.RunCommand(ctx.Client, psCmd, os.Stdout, os.Stderr)
			if err != nil {
				return fmt.Errorf("status (compose ps via SSH): %w", err)
			}
			if code != 0 {
				fmt.Fprintf(os.Stderr, "warning: compose ps exited with code %d\n", code)
			}
		}

		if mode != ModeProxy {
			return nil
		}

		pf, pfErr := config.LoadProjectFile(config.ProjectFileName)
		if pfErr != nil || pf.Validate() != nil {
			// JSON consumers need a deterministic failure; table consumers
			// have already seen compose ps and can live without proxy detail.
			if format == "json" {
				if pfErr == nil {
					pfErr = pf.Validate()
				}
				return fmt.Errorf("load conoha.yml: %w", pfErr)
			}
			return nil
		}

		dataDir, _ := cmd.Flags().GetString("data-dir")
		if dataDir == "" {
			dataDir = proxy.DefaultDataDir
		}
		admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))

		report, err := collectAppStatus(admin, pf, os.Stderr)
		if err != nil {
			if format == "json" {
				return err
			}
			fmt.Fprintf(os.Stderr, "\n==> Proxy service %q: (error: %v)\n", pf.Name, err)
			return nil
		}

		if format == "json" {
			return renderStatusJSON(os.Stdout, report)
		}
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "==> Proxy services")
		return renderStatusTable(os.Stdout, report)
	},
}

// collectAppStatus fetches root + per-expose proxy services. A Get failure on
// the root service is fatal (it's the primary target). A Get failure on any
// expose service is logged as a warning and recorded as a nil Service entry
// so the other rows still render.
func collectAppStatus(admin statusClient, pf *config.ProjectFile, warn io.Writer) (*appStatusReport, error) {
	root, err := admin.Get(pf.Name)
	if err != nil {
		return nil, fmt.Errorf("proxy get %q: %w", pf.Name, err)
	}
	r := &appStatusReport{Root: root}
	for i := range pf.Expose {
		b := &pf.Expose[i]
		name := exposeServiceName(pf.Name, b.Label)
		svc, gErr := admin.Get(name)
		if gErr != nil {
			_, _ = fmt.Fprintf(warn, "warning: proxy get %s: %v\n", name, gErr)
			r.Expose = append(r.Expose, exposeStatusEntry{Label: b.Label})
			continue
		}
		r.Expose = append(r.Expose, exposeStatusEntry{Label: b.Label, Service: svc})
	}
	return r, nil
}

// renderStatusTable writes the report as a padded TARGET/HOST/PHASE/ACTIVE/
// DRAIN DEADLINE/TLS table. `web` is the fixed label for the root service;
// expose rows use their block label.
func renderStatusTable(w io.Writer, r *appStatusReport) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "TARGET\tHOST\tPHASE\tACTIVE\tDRAIN DEADLINE\tTLS")
	writeStatusRow(tw, "web", r.Root)
	for _, e := range r.Expose {
		writeStatusRow(tw, e.Label, e.Service)
	}
	return tw.Flush()
}

func writeStatusRow(w io.Writer, target string, svc *proxypkg.Service) {
	host, phase, active, deadline, tls := "-", "-", "-", "-", "-"
	if svc != nil {
		if len(svc.Hosts) > 0 {
			host = strings.Join(svc.Hosts, ",")
		}
		if svc.Phase != "" {
			phase = string(svc.Phase)
		}
		if svc.ActiveTarget != nil {
			active = svc.ActiveTarget.URL
		}
		if svc.DrainDeadline != nil {
			deadline = svc.DrainDeadline.Format(time.RFC3339)
		}
		if svc.TLSStatus != "" {
			tls = svc.TLSStatus
		}
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", target, host, phase, active, deadline, tls)
}

func renderStatusJSON(w io.Writer, r *appStatusReport) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}

func buildStatusCmdForProxy(app string) string {
	// Enumerate slot projects via container labels rather than
	// 'docker compose ls --format "{{.Name}}"', which fails silently on
	// Docker Compose v5 hosts and would produce an empty listing (#114).
	return fmt.Sprintf(
		`for p in $(%[1]s); do `+
			`echo "--- compose project: ${p} ---"; `+
			`docker compose -p "${p}" ps; `+
			`done`,
		composeProjectEnumPipeline(app))
}

func buildStatusCmdForNoProxy(app string) string {
	return fmt.Sprintf("cd /opt/conoha/%s && docker compose ps", app)
}
