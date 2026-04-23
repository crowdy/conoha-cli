package app

import (
	"strings"
	"testing"
)

func TestDeployCmd_HasModeFlags(t *testing.T) {
	if deployCmd.Flags().Lookup("proxy") == nil {
		t.Error("deploy should have --proxy flag")
	}
	if deployCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("deploy should have --no-proxy flag")
	}
	if deployCmd.Flags().Lookup("app-name") == nil {
		t.Error("deploy should have --app-name flag (required with --no-proxy)")
	}
}

func TestBuildNoProxyDeployCmd(t *testing.T) {
	got := buildNoProxyDeployCmd("/opt/conoha/myapp", "myapp", "compose.yml")
	for _, want := range []string{
		"cd '/opt/conoha/myapp'",
		"docker compose -p myapp",
		"-f 'compose.yml'",
		"up -d --build",
		// .env.server merged into .env: new canonical path (v0.2+) preferred,
		// legacy path used as read fallback with a deprecation warning.
		"/opt/conoha/myapp/.env.server",
		"/opt/conoha/myapp.env.server",
		"warning: merging legacy env file",
		">> .env",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildNoProxyUploadCmd(t *testing.T) {
	got := buildNoProxyUploadCmd("/opt/conoha/myapp")
	for _, want := range []string{
		"mkdir -p '/opt/conoha/myapp'",
		"tar xzf - -C '/opt/conoha/myapp'",
		// Remove previous deploy's merged .env so tar becomes authoritative
		// for repo-level env and `app env unset` takes effect on redeploy (C1 fix).
		"rm -f '/opt/conoha/myapp/.env'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	// Must not wipe the entire app dir (would destroy named volumes + env.server dir siblings).
	if strings.Contains(got, "rm -rf '/opt/conoha/myapp'") {
		t.Errorf("no-proxy upload must not wipe app dir: %s", got)
	}
}
