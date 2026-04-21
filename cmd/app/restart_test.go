package app

import (
	"strings"
	"testing"
)

func TestRestartCmd_HasModeFlags(t *testing.T) {
	if restartCmd.Flags().Lookup("proxy") == nil {
		t.Error("restart should have --proxy flag")
	}
	if restartCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("restart should have --no-proxy flag")
	}
}

func TestBuildRestartCmd_Proxy(t *testing.T) {
	got := buildRestartCmdForProxy("myapp", "abc1234")
	for _, want := range []string{
		"docker compose -p myapp-abc1234 restart",
		"docker compose -p myapp-abc1234 ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildRestartCmd_NoProxy(t *testing.T) {
	got := buildRestartCmdForNoProxy("myapp")
	for _, want := range []string{
		"cd /opt/conoha/myapp",
		"docker compose restart",
		"docker compose ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
