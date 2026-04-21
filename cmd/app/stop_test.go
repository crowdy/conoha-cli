package app

import (
	"strings"
	"testing"
)

func TestStopCmd_HasModeFlags(t *testing.T) {
	if stopCmd.Flags().Lookup("proxy") == nil {
		t.Error("stop should have --proxy flag")
	}
	if stopCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("stop should have --no-proxy flag")
	}
}

func TestBuildStopCmd_Proxy(t *testing.T) {
	got := buildStopCmdForProxy("myapp", "abc1234")
	for _, want := range []string{
		"docker compose -p myapp-abc1234 stop",
		"docker compose -p myapp-abc1234 ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildStopCmd_NoProxy(t *testing.T) {
	got := buildStopCmdForNoProxy("myapp")
	for _, want := range []string{
		"cd /opt/conoha/myapp",
		"docker compose stop",
		"docker compose ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
