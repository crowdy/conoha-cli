package app

import (
	"strings"
	"testing"
)

func TestStatusCmd_HasModeFlags(t *testing.T) {
	if statusCmd.Flags().Lookup("proxy") == nil {
		t.Error("status should have --proxy flag")
	}
	if statusCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("status should have --no-proxy flag")
	}
}

func TestBuildStatusCmd_Proxy(t *testing.T) {
	got := buildStatusCmdForProxy("myapp")
	for _, want := range []string{
		"docker compose ls",
		`grep -E "^myapp(-|$)"`,
		"docker compose -p",
		"ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildStatusCmd_NoProxy(t *testing.T) {
	got := buildStatusCmdForNoProxy("myapp")
	for _, want := range []string{
		"cd /opt/conoha/myapp",
		"docker compose ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
