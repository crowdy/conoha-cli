package app

import (
	"errors"
	"fmt"
	"io"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

// proxyAdmin is the subset of *proxypkg.Client used by init/destroy. It lets
// tests substitute a fake without touching SSH.
type proxyAdmin interface {
	Upsert(req proxypkg.UpsertRequest) (*proxypkg.Service, error)
	Delete(name string) error
}

// exposeServiceName returns the proxy service name for an expose block,
// matching the Q-naming convention in the subdomain-split RFC (§7).
func exposeServiceName(appName, label string) string {
	return appName + "-" + label
}

// healthFor returns the proxy health policy a block should be registered with.
// Each expose block may override the top-level health; if not, the root health
// is inherited.
func healthFor(pf *config.ProjectFile, block *config.ExposeBlock) *proxypkg.HealthPolicy {
	if block != nil && block.Health != nil {
		return mapHealth(block.Health)
	}
	return mapHealth(pf.Health)
}

// registerProxyServices upserts one proxy service for the root web target
// plus one per expose block. If any upsert fails, every service registered
// earlier in this call is deleted (best-effort) before the original error
// is returned, so a partial failure cannot leave orphan services behind.
//
// Registration order: root first, then exposes in declaration order. Rollback
// unwinds in reverse (expose[n-1] … expose[0] … root).
func registerProxyServices(admin proxyAdmin, pf *config.ProjectFile, log io.Writer) error {
	registered := make([]string, 0, 1+len(pf.Expose))

	rootSvc, err := admin.Upsert(proxypkg.UpsertRequest{
		Name:         pf.Name,
		Hosts:        pf.Hosts,
		HealthPolicy: mapHealth(pf.Health),
	})
	if err != nil {
		return err
	}
	registered = append(registered, pf.Name)
	_, _ = fmt.Fprintf(log, "Service %q registered. phase=%s tls=%s\n", rootSvc.Name, rootSvc.Phase, rootSvc.TLSStatus)

	for i := range pf.Expose {
		b := &pf.Expose[i]
		name := exposeServiceName(pf.Name, b.Label)
		svc, upErr := admin.Upsert(proxypkg.UpsertRequest{
			Name:         name,
			Hosts:        []string{b.Host},
			HealthPolicy: healthFor(pf, b),
		})
		if upErr != nil {
			for j := len(registered) - 1; j >= 0; j-- {
				if delErr := admin.Delete(registered[j]); delErr != nil && !errors.Is(delErr, proxypkg.ErrNotFound) {
					_, _ = fmt.Fprintf(log, "warning: rollback delete %s: %v\n", registered[j], delErr)
				}
			}
			return fmt.Errorf("upsert expose %q (%s): %w", b.Label, b.Host, upErr)
		}
		registered = append(registered, name)
		_, _ = fmt.Fprintf(log, "==> Registered %q on %s (phase=%s tls=%s)\n", svc.Name, b.Host, svc.Phase, svc.TLSStatus)
	}
	return nil
}

// deregisterProxyServices deletes the expose services (in reverse declaration
// order) then the root service. 404s are tolerated; other errors are logged
// as warnings but never abort the sweep — destroy has already removed the
// app's on-server state, so a leftover proxy registration is the lesser evil
// compared to a half-cleaned app.
func deregisterProxyServices(admin proxyAdmin, pf *config.ProjectFile, log io.Writer) {
	for i := len(pf.Expose) - 1; i >= 0; i-- {
		name := exposeServiceName(pf.Name, pf.Expose[i].Label)
		if err := admin.Delete(name); err != nil && !errors.Is(err, proxypkg.ErrNotFound) {
			_, _ = fmt.Fprintf(log, "warning: proxy delete %s: %v\n", name, err)
		} else if err == nil {
			_, _ = fmt.Fprintf(log, "==> Deregistered %q from proxy\n", name)
		}
	}
	if err := admin.Delete(pf.Name); err != nil && !errors.Is(err, proxypkg.ErrNotFound) {
		_, _ = fmt.Fprintf(log, "warning: proxy delete %s: %v\n", pf.Name, err)
	} else if err == nil {
		_, _ = fmt.Fprintf(log, "==> Deregistered %q from proxy\n", pf.Name)
	}
}
