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
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "rm -rf '/opt/conoha/myapp'") {
		t.Errorf("no-proxy upload must not wipe app dir: %s", got)
	}
}
