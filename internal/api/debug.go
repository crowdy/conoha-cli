package api

import (
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
			fmt.Fprintf(os.Stderr, "> %s\n", maskSensitive(string(body)))
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
			fmt.Fprintf(os.Stderr, "< %s\n", maskSensitive(string(body)))
		}
	}
	fmt.Fprintln(os.Stderr)
}
