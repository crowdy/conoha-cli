package app

import (
	"strings"
	"testing"
)

func TestLogsCmd_HasModeFlags(t *testing.T) {
	if logsCmd.Flags().Lookup("proxy") == nil {
		t.Error("logs should have --proxy flag")
	}
	if logsCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("logs should have --no-proxy flag")
	}
}

func TestBuildLogsCmd_Proxy(t *testing.T) {
	got := buildLogsCmdForProxy("myapp", "abc1234", 100, false, "")
	for _, want := range []string{
		"docker compose -p myapp-abc1234",
		"logs",
		"--tail 100",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildLogsCmd_Proxy_FollowService(t *testing.T) {
	got := buildLogsCmdForProxy("myapp", "abc1234", 50, true, "web")
	for _, want := range []string{
		"docker compose -p myapp-abc1234 logs",
		"--tail 50",
		"-f",
		" web",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildLogsCmd_NoProxy(t *testing.T) {
	got := buildLogsCmdForNoProxy("myapp", 100, false, "")
	for _, want := range []string{
		"cd /opt/conoha/myapp",
		"docker compose logs",
		"--tail 100",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
