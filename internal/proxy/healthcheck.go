package proxy

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"
)

// Sleeper is `time.Sleep` in production. Tests inject a no-op or fast-forward
// implementation so polling-loop assertions don't depend on wall-clock.
type Sleeper func(time.Duration)

// HealthcheckOptions tunes WaitForHealthy. PollInterval defaults to 1s,
// StableSamples (consecutive "running" samples required before checking
// /healthz) defaults to 3.
type HealthcheckOptions struct {
	PollInterval  time.Duration
	StableSamples int
}

// WaitForHealthy polls the proxy container until both conditions hold:
//
//  1. `docker inspect --format '{{.State.Status}}' <container>` returns
//     `running` for at least StableSamples consecutive samples (defaults
//     to 3). Anything else — `restarting`, `exited`, missing — resets the
//     counter, so a crash loop doesn't false-positive on its brief running
//     windows.
//  2. `curl --unix-socket <dataDir>/admin.sock http://admin/healthz` returns
//     `{"status":"ok"}`. Polled only after the stability gate trips, since
//     the proxy can't serve healthz until it's bound.
//
// On timeout the error includes the last 20 lines of `docker logs` so the
// operator doesn't need a manual SSH round-trip to diagnose. This is the
// gap that let the bad #172 fix slip through CI: `docker run -d` returned
// 0 even though the container was crash-looping with "permission denied"
// on bind. Folding the check into proxy boot itself means same-class
// regressions (file caps, sysctl drift, image entrypoint changes) fail
// loudly on first run rather than only during a manual log read.
func WaitForHealthy(exec Executor, container, dataDir string, timeout time.Duration, sleep Sleeper, opt HealthcheckOptions) error {
	if sleep == nil {
		sleep = time.Sleep
	}
	if opt.PollInterval <= 0 {
		opt.PollInterval = 1 * time.Second
	}
	if opt.StableSamples <= 0 {
		opt.StableSamples = 3
	}
	deadline := time.Now().Add(timeout)
	healthy := 0
	lastStatus := "unknown"
	for {
		status := dockerInspectStatus(exec, container)
		lastStatus = status
		if status == "running" {
			healthy++
			if healthy >= opt.StableSamples && healthzOK(exec, dataDir) {
				return nil
			}
		} else {
			healthy = 0
		}
		if !time.Now().Before(deadline) {
			break
		}
		sleep(opt.PollInterval)
	}
	logs := dockerLogsTail(exec, container, 20)
	return fmt.Errorf("conoha-proxy did not become healthy within %s (last container status: %q). "+
		"Last %d lines of `docker logs %s`:\n%s",
		timeout, lastStatus, 20, container, indentLines(logs, "  "))
}

// dockerInspectStatus returns the container's State.Status, or "missing" when
// the container does not exist. We intentionally don't propagate the inspect
// error: from the caller's perspective "container exited" and "container was
// never created" both mean "not running" and the polling loop reacts
// identically.
func dockerInspectStatus(exec Executor, container string) string {
	var out bytes.Buffer
	cmd := fmt.Sprintf("docker inspect --format '{{.State.Status}}' %s 2>/dev/null", shellQuote(container))
	if err := exec.Run(cmd, nil, &out); err != nil {
		return "missing"
	}
	return strings.TrimSpace(out.String())
}

// healthzOK returns true iff the admin socket responds with the expected
// {"status":"ok"} JSON. A connection error, non-200, or any other body is
// treated as "not yet healthy" so the caller can keep polling.
func healthzOK(exec Executor, dataDir string) bool {
	var out bytes.Buffer
	cmd := fmt.Sprintf("curl -sf --unix-socket %s/admin.sock http://admin/healthz", dataDir+"")
	if err := exec.Run(cmd, nil, &out); err != nil {
		return false
	}
	return strings.Contains(out.String(), `"status":"ok"`)
}

func dockerLogsTail(exec Executor, container string, lines int) string {
	var out bytes.Buffer
	cmd := fmt.Sprintf("docker logs --tail %d %s 2>&1", lines, shellQuote(container))
	_ = exec.Run(cmd, nil, &out)
	return out.String()
}

func indentLines(s, prefix string) string {
	if s == "" {
		return ""
	}
	var out strings.Builder
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		out.WriteString(prefix)
		out.WriteString(line)
		out.WriteString("\n")
	}
	return out.String()
}

// shellQuote escapes a string for safe single-quoted use on the remote shell.
// Container names and data-dir paths are already validated at the flag layer,
// but quoting keeps us safe if those constraints ever relax.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// Compile-time guard: WaitForHealthy must accept an io.Writer-friendly Executor.
// Keeps refactors of the proxy.Executor signature from silently breaking us.
var _ = func() Executor {
	type _exec interface {
		Run(string, io.Reader, io.Writer) error
	}
	var e _exec
	if e == nil {
		return nil
	}
	return e.(Executor)
}()
