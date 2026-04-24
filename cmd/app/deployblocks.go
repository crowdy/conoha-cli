package app

import (
	"github.com/crowdy/conoha-cli/internal/config"
)

// DeployBlock is the unified per-target record that runProxyDeployState
// iterates over — root web plus every blue/green expose block. It decouples
// the state machine from ProjectFile field plumbing so a single loop handles
// N targets.
type DeployBlock struct {
	// Label distinguishes this block for log lines and proxy service naming.
	// Empty for the root web block.
	Label string
	// Service is the compose service name to bring up in the slot. It maps
	// to container_name "<app>-<slot>-<service>" via the compose override.
	Service string
	// Port is the container-internal port the service listens on.
	Port int
	// Host is the public host the proxy routes to this block. Only used for
	// human-facing log output; the proxy service registration already carries
	// authoritative host list (phase 2).
	Host string
	// ProxyName is the proxy service name: "<app>" for root, "<app>-<label>"
	// for an expose block. Identifies the /deploy and /rollback target.
	ProxyName string
}

// isRoot reports whether this block is the root web target (Label is empty).
func (b DeployBlock) isRoot() bool { return b.Label == "" }

// collectDeployBlocks returns the blocks that participate in blue/green
// rotation on a deploy: the root web, plus every expose block whose
// BlueGreen is nil (default-true) or explicitly true. Expose blocks with
// BlueGreen == false are handled by collectEffectiveAccessories instead —
// they go up once in the shared accessory compose project and never rotate.
//
// Order: root first in the slice, expose blocks in declaration order.
// runProxyDeployState iterates /deploy the other direction (expose first,
// root last) so callers must explicitly reorder at that boundary.
func collectDeployBlocks(pf *config.ProjectFile) []DeployBlock {
	out := make([]DeployBlock, 0, 1+len(pf.Expose))
	out = append(out, DeployBlock{
		Service:   pf.Web.Service,
		Port:      pf.Web.Port,
		ProxyName: pf.Name,
	})
	for i := range pf.Expose {
		b := &pf.Expose[i]
		if b.BlueGreen != nil && !*b.BlueGreen {
			continue
		}
		out = append(out, DeployBlock{
			Label:     b.Label,
			Service:   b.Service,
			Port:      b.Port,
			Host:      b.Host,
			ProxyName: exposeServiceName(pf.Name, b.Label),
		})
	}
	return out
}

// collectEffectiveAccessories returns the set of compose services that
// should come up in the persistent accessory project: the explicit
// pf.Accessories list plus any expose block whose BlueGreen is explicitly
// false. BlueGreen-false blocks declare a public host but are started once
// (accessory-style) and never rotate with a slot.
func collectEffectiveAccessories(pf *config.ProjectFile) []string {
	out := make([]string, 0, len(pf.Accessories))
	out = append(out, pf.Accessories...)
	for i := range pf.Expose {
		b := &pf.Expose[i]
		if b.BlueGreen != nil && !*b.BlueGreen {
			out = append(out, b.Service)
		}
	}
	return out
}
