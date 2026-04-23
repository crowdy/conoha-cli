package proxy

import (
	"strings"
	"testing"
)

func TestBootScript_ContainsEssentials(t *testing.T) {
	s := string(BootScript(BootParams{
		Email:     "ops@example.com",
		Image:     "ghcr.io/crowdy/conoha-proxy:latest",
		DataDir:   "/var/lib/conoha-proxy",
		Container: "conoha-proxy",
	}))
	for _, want := range []string{
		"set -euo pipefail",
		"mkdir -p /var/lib/conoha-proxy",
		"chown 65532:65532 /var/lib/conoha-proxy",
		"--network host",
		"-v /var/lib/conoha-proxy:/var/lib/conoha-proxy",
		"ghcr.io/crowdy/conoha-proxy:latest",
		"--acme-email=ops@example.com",
		"--name conoha-proxy",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("BootScript missing %q:\n%s", want, s)
		}
	}
}

func TestRebootScript_PullsStopsRemovesStarts(t *testing.T) {
	s := string(RebootScript(BootParams{
		Email:     "ops@example.com",
		Image:     "ghcr.io/crowdy/conoha-proxy:latest",
		DataDir:   "/var/lib/conoha-proxy",
		Container: "conoha-proxy",
	}))
	for _, want := range []string{
		"docker pull ghcr.io/crowdy/conoha-proxy:latest",
		"docker stop conoha-proxy",
		"docker rm conoha-proxy",
		"--network host",
		"--acme-email=ops@example.com",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("RebootScript missing %q:\n%s", want, s)
		}
	}
}

func TestSimpleScripts(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"start", string(StartScript("conoha-proxy")), "docker start conoha-proxy"},
		{"stop", string(StopScript("conoha-proxy")), "docker stop conoha-proxy"},
		{"restart", string(RestartScript("conoha-proxy")), "docker restart conoha-proxy"},
	}
	for _, tc := range cases {
		if !strings.Contains(tc.got, tc.want) {
			t.Errorf("%s script missing %q:\n%s", tc.name, tc.want, tc.got)
		}
	}
}

func TestRemoveScript_Purge(t *testing.T) {
	s := string(RemoveScript("conoha-proxy", "/var/lib/conoha-proxy", true))
	if !strings.Contains(s, "docker rm -f conoha-proxy") {
		t.Errorf("missing rm: %s", s)
	}
	if !strings.Contains(s, "rm -rf /var/lib/conoha-proxy") {
		t.Errorf("missing purge: %s", s)
	}

	s = string(RemoveScript("conoha-proxy", "/var/lib/conoha-proxy", false))
	if strings.Contains(s, "rm -rf /var/lib/conoha-proxy") {
		t.Errorf("non-purge should NOT delete data dir: %s", s)
	}
}

func TestLogsScript_FollowAndLines(t *testing.T) {
	s := string(LogsScript("conoha-proxy", true, 50))
	if !strings.Contains(s, "-f") || !strings.Contains(s, "--tail 50") {
		t.Errorf("flags missing: %s", s)
	}
	s = string(LogsScript("conoha-proxy", false, 0))
	if strings.Contains(s, "-f") || strings.Contains(s, "--tail") {
		t.Errorf("no flags expected: %s", s)
	}
}
