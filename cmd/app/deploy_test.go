package app

import (
	"strings"
	"testing"
)

func TestBuildUploadCmd_EnvServerPathInsideWorkDir(t *testing.T) {
	// Bug #81: ENV_FILE was "/opt/conoha/<name>.env.server" (sibling of workDir)
	// Correct:  "/opt/conoha/<name>/.env.server" (inside workDir)
	cmd := buildUploadCmd("/opt/conoha/myapp")
	if strings.Contains(cmd, "/opt/conoha/myapp.env.server") {
		t.Error("buildUploadCmd looks for .env.server as sibling of workDir (bug #81); must look inside workDir")
	}
	if !strings.Contains(cmd, "/opt/conoha/myapp/.env.server") {
		t.Error("buildUploadCmd must look for .env.server inside workDir (#81)")
	}
}

func TestBuildUploadCmd_UsesExistenceSentinelNotContentCheck(t *testing.T) {
	// Bug #85 edge case: an empty .env file must also be preserved after redeploy.
	// Using [ -n "$ENV_BACKUP" ] (content check) would silently drop a zero-byte .env
	// because command substitution strips trailing newlines and empty content is falsy.
	// The fix uses a boolean ENV_EXISTS sentinel instead.
	cmd := buildUploadCmd("/opt/conoha/myapp")
	if !strings.Contains(cmd, "ENV_EXISTS") {
		t.Error("buildUploadCmd must use ENV_EXISTS sentinel to detect pre-existing .env (#85)")
	}
	if strings.Contains(cmd, `[ -n "$ENV_BACKUP" ]`) {
		t.Error("buildUploadCmd must not gate restore on content; empty .env would be silently lost")
	}
}

func TestBuildComposeCmd_PassesEnvFileForBuildArgs(t *testing.T) {
	// Issue #82: NEXT_PUBLIC_* vars need --env-file at docker compose build time
	cmd := buildComposeCmd("/opt/conoha/myapp", "compose.yml")
	if !strings.Contains(cmd, "--env-file") {
		t.Error("buildComposeCmd must pass --env-file so Docker build args pick up .env vars (#82)")
	}
}
