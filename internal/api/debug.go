package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/crowdy/conoha-cli/internal/config"
)

// DebugLevel controls the verbosity of debug output.
type DebugLevel int

const (
	DebugOff     DebugLevel = 0
	DebugVerbose DebugLevel = 1 // method, URL, status, duration
	DebugAPI     DebugLevel = 2 // + headers, bodies
)

var debugLevel DebugLevel

func init() {
	switch os.Getenv(config.EnvDebug) {
	case "api":
		debugLevel = DebugAPI
	case "1", "true":
		debugLevel = DebugVerbose
	}
}

// SetDebugLevel sets the debug level. Only increases (never decreases).
func SetDebugLevel(level DebugLevel) {
	if level > debugLevel {
		debugLevel = level
	}
}

// sensitiveHeaders are headers whose values should be masked.
var sensitiveHeaders = map[string]bool{
	"X-Auth-Token":    true,
	"X-Subject-Token": true,
	"Authorization":   true,
}

var passwordRe = regexp.MustCompile(`"password"\s*:\s*"[^"]*"`)

// maskSensitive masks passwords and tokens in a string.
func maskSensitive(s string) string {
	return passwordRe.ReplaceAllString(s, `"password":"****"`)
}

// formatBody renders an HTTP body for debug output. JSON is pretty-printed and
// every line is prefixed with dir ("> " for requests, "< " for responses) so a
// multi-kilobyte payload prints as many short, readable lines rather than one
// giant line whose start scrolls off-screen. Non-JSON bodies are emitted
// verbatim on a single prefixed line. Passwords are always masked.
func formatBody(dir string, body []byte) string {
	masked := maskSensitive(string(body))
	var pretty bytes.Buffer
	if json.Indent(&pretty, []byte(masked), "", "  ") == nil {
		masked = pretty.String()
	}
	var b strings.Builder
	// Trim a single trailing newline so a body that already ends in "\n"
	// (common for non-JSON payloads) does not yield a bare prefix-only line.
	for _, line := range strings.Split(strings.TrimSuffix(masked, "\n"), "\n") {
		b.WriteString(dir)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func debugLogRequest(req *http.Request, body []byte) {
	if debugLevel < DebugVerbose {
		return
	}
	fmt.Fprintf(os.Stderr, "> %s %s\n", req.Method, req.URL.String())
	if debugLevel >= DebugAPI {
		for name, values := range req.Header {
			val := strings.Join(values, ", ")
			if sensitiveHeaders[name] {
				val = "****"
			}
			fmt.Fprintf(os.Stderr, "> %s: %s\n", name, val)
		}
		if len(body) > 0 {
			fmt.Fprint(os.Stderr, formatBody("> ", body))
		}
	}
}

func debugLogResponse(resp *http.Response, duration time.Duration, body []byte) {
	if debugLevel < DebugVerbose {
		return
	}
	fmt.Fprintf(os.Stderr, "< %d %s (%dms)\n", resp.StatusCode, http.StatusText(resp.StatusCode), duration.Milliseconds())
	if debugLevel >= DebugAPI {
		for name, values := range resp.Header {
			val := strings.Join(values, ", ")
			if sensitiveHeaders[name] {
				val = "****"
			}
			fmt.Fprintf(os.Stderr, "< %s: %s\n", name, val)
		}
		if len(body) > 0 {
			fmt.Fprint(os.Stderr, formatBody("< ", body))
		}
	}
	fmt.Fprintln(os.Stderr)
}
