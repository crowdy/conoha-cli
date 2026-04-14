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

func TestBuildUploadCmd_BacksUpEnvBeforeRmRf(t *testing.T) {
	// Bug #85: rm -rf deletes .env before extraction; existing .env must be preserved
	cmd := buildUploadCmd("/opt/conoha/myapp")
	if !strings.Contains(cmd, "ENV_BACKUP") {
		t.Error("buildUploadCmd must backup existing .env before rm -rf (#85)")
	}
}

func TestBuildComposeCmd_PassesEnvFileForBuildArgs(t *testing.T) {
	// Issue #82: NEXT_PUBLIC_* vars need --env-file at docker compose build time
	cmd := buildComposeCmd("/opt/conoha/myapp", "compose.yml")
	if !strings.Contains(cmd, "--env-file") {
		t.Error("buildComposeCmd must pass --env-file so Docker build args pick up .env vars (#82)")
	}
}
