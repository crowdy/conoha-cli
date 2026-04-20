package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// buildSlotUploadCmd extracts the incoming tar archive into the slot-specific
// work directory. .env preservation from the old single-slot flow is NOT
// applied here — env handling belongs to accessories or app.env commands.
func buildSlotUploadCmd(workDir, _ string) string {
	return fmt.Sprintf(
		"rm -rf '%[1]s' && mkdir -p '%[1]s' && tar xzf - -C '%[1]s'",
		workDir)
}

// buildSlotComposeUp starts the single web service inside a slot-scoped project.
func buildSlotComposeUp(workDir, project, composeFile, overrideFile, webService string) string {
	return fmt.Sprintf(
		"cd '%s' && docker compose -p %s -f %s -f %s up -d --build %s",
		workDir, project, composeFile, overrideFile, webService)
}

// buildDockerPortCmd produces a command that prints the host:port mapping
// for the web container's internal port.
func buildDockerPortCmd(containerName string, port int) string {
	return fmt.Sprintf("docker port %s %d", containerName, port)
}

var hostPortRe = regexp.MustCompile(`(?m)^(?:\d+\.\d+\.\d+\.\d+|\[::\]|\[::1\]):(\d+)`)

// extractHostPort parses "docker port" output and returns the first loopback
// (127.0.0.1) mapping if present, otherwise the first IPv4/IPv6 mapping found.
// Returns an error if no mapping line exists.
func extractHostPort(out string) (int, error) {
	// Prefer 127.0.0.1 lines explicitly.
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "127.0.0.1:") {
			return parseColonPort(line)
		}
	}
	m := hostPortRe.FindStringSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("no host port in docker port output: %q", out)
	}
	return strconv.Atoi(m[1])
}

func parseColonPort(line string) (int, error) {
	line = strings.TrimSpace(line)
	i := strings.LastIndex(line, ":")
	if i < 0 {
		return 0, fmt.Errorf("no colon in %q", line)
	}
	return strconv.Atoi(line[i+1:])
}

// buildScheduleDrainCmd fires a detached shell that, after drainMs, brings the
// old slot down with `docker compose down`. Uses nohup + background shell;
// does not rely on `at` availability.
func buildScheduleDrainCmd(workDir, project string, drainMs int) string {
	seconds := drainMs / 1000
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf(
		`nohup bash -c "sleep %d && cd '%s' && docker compose -p %s down" >/dev/null 2>&1 & disown`,
		seconds, workDir, project)
}

// buildAccessoryUp starts the accessories listed, using a dedicated compose
// project so they survive slot teardown. WorkDir is the slot's work directory —
// we read the compose file from there because accessories share the same file.
func buildAccessoryUp(workDir, project, composeFile string, accessories []string) string {
	args := strings.Join(accessories, " ")
	return fmt.Sprintf(
		"cd '%s' && docker compose -p %s -f %s up -d %s",
		workDir, project, composeFile, args)
}

// buildAccessoryExists reports (via shell exit 0/1) whether the accessory project
// has any containers. Exit 0 means "already up".
func buildAccessoryExists(project string) string {
	return fmt.Sprintf(
		`[ "$(docker compose -p %s ps -q | wc -l)" -gt 0 ]`,
		project)
}
