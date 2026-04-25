package app

import (
	"fmt"
	"strings"
)

// composeOverrideFor returns a compose override document (YAML) with one
// entry per blue/green DeployBlock. Each entry:
//   - pins container name to <app>-<slot>-<service>
//   - maps 127.0.0.1:0:<port> so the kernel picks a free host port
//   - attaches env_file: /opt/conoha/<app>/.env.server
//   - when hasAccessories, joins the <app>-accessories_default network
//
// Per RFC §7 Q-env the env_file is shared across all blocks; a later RFC
// will introduce per-block overrides if needed.
//
// For backward compatibility with phase 2 and earlier, a single-block call
// (root web only, no expose entries) produces byte-identical output to the
// pre-phase-3 composeOverride.
func composeOverrideFor(app, slot string, blocks []DeployBlock, hasAccessories bool) string {
	var sb strings.Builder
	sb.WriteString("services:\n")
	for _, b := range blocks {
		fmt.Fprintf(&sb, "  %s:\n", b.Service)
		fmt.Fprintf(&sb, "    container_name: %s-%s-%s\n", app, slot, b.Service)
		sb.WriteString("    ports:\n")
		fmt.Fprintf(&sb, "      - \"127.0.0.1:0:%d\"\n", b.Port)
		sb.WriteString("    env_file:\n")
		fmt.Fprintf(&sb, "      - /opt/conoha/%s/.env.server\n", app)
		if hasAccessories {
			sb.WriteString("    networks:\n")
			sb.WriteString("      - default\n")
			sb.WriteString("      - accessories\n")
		}
	}
	if hasAccessories {
		sb.WriteString("networks:\n")
		sb.WriteString("  accessories:\n")
		fmt.Fprintf(&sb, "    name: %s-accessories_default\n", app)
		sb.WriteString("    external: true\n")
	}
	return sb.String()
}

// composeOverride is the legacy single-web-service form. Preserved so call
// sites outside runProxyDeployState (and the pre-existing gold tests) need
// no change.
func composeOverride(app, slot, webService string, webPort int, hasAccessories bool) string {
	return composeOverrideFor(app, slot, []DeployBlock{{Service: webService, Port: webPort}}, hasAccessories)
}

// composeOverrideForAccessories returns a compose override (YAML) that
// publishes a host port and attaches .env.server for each fixed expose
// block (BlueGreen:false). Returns empty string when there are no fixed
// blocks, so the caller can skip writing/passing the file.
//
// Unlike composeOverrideFor, this override does NOT pin a slot-scoped
// container_name (these containers are slot-agnostic and live in the
// stable accessory project) and does NOT join the accessories network
// (they ARE in that project's default network already).
func composeOverrideForAccessories(app string, fixedBlocks []DeployBlock) string {
	if len(fixedBlocks) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("services:\n")
	for _, b := range fixedBlocks {
		fmt.Fprintf(&sb, "  %s:\n", b.Service)
		sb.WriteString("    ports:\n")
		fmt.Fprintf(&sb, "      - \"127.0.0.1:0:%d\"\n", b.Port)
		sb.WriteString("    env_file:\n")
		fmt.Fprintf(&sb, "      - /opt/conoha/%s/.env.server\n", app)
	}
	return sb.String()
}

// slotProjectName is the compose -p value for a blue/green slot.
func slotProjectName(app, slot string) string {
	return fmt.Sprintf("%s-%s", app, slot)
}

// accessoryProjectName is the compose -p value for the persistent accessory stack.
func accessoryProjectName(app string) string {
	return app + "-accessories"
}
