package app

import "fmt"

// composeProjectEnumPipeline returns a bash pipeline that prints the names of
// all Compose projects whose name equals appName or starts with
// "appName-" — this matches both no-proxy deployments (project = app) and
// proxy slot deployments (project = app-blue / app-green).
//
// The enumeration uses the com.docker.compose.project container label via
// `docker ps -a` rather than `docker compose ls --format '{{.Name}}'`.
// Docker Compose v5 removed Go-template support for `compose ls --format`
// (only table/json are accepted), so the template form silently fails on
// recent hosts — see issue #114. Labels are stable across versions.
//
// appName is interpolated verbatim into the generated shell text.
// Callers may pass either:
//   - a Go string holding an already-validated app name — cmd/app loaders
//     (see cmd/app/connect.go) enforce the [A-Za-z0-9_-] charset via
//     internalssh.ValidateAppName before the context reaches this helper; or
//   - a bash variable reference such as "${APP_NAME}" to defer expansion
//     to the remote shell. In that form the caller is responsible for
//     ensuring the variable holds a validated name at expansion time
//     (destroy.go does this: APP_NAME is set from a Go-validated value
//     one line above the pipeline).
func composeProjectEnumPipeline(appName string) string {
	return fmt.Sprintf(
		`docker ps -a --format '{{.Label "com.docker.compose.project"}}' 2>/dev/null | awk 'NF' | sort -u | grep -E "^%s(-|$)" || true`,
		appName,
	)
}
