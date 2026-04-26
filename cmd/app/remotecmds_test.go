package app

import (
	"strings"
	"testing"
)

func TestBuildSlotUploadCmd(t *testing.T) {
	got := buildSlotUploadCmd("/opt/conoha/myapp/abc1234", "127.0.0.1")
	for _, want := range []string{
		"rm -rf '/opt/conoha/myapp/abc1234'",
		"mkdir -p '/opt/conoha/myapp/abc1234'",
		"tar xzf - -C '/opt/conoha/myapp/abc1234'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildComposeUp_Slot(t *testing.T) {
	got := buildSlotComposeUp("/opt/conoha/myapp/abc1234", "myapp-abc1234", "compose.yml", "override.yml", "myapp", []string{"web"})
	for _, want := range []string{
		"cd '/opt/conoha/myapp/abc1234'",
		"touch '/opt/conoha/myapp/.env.server'",
		"docker compose --env-file '/opt/conoha/myapp/.env.server' -p myapp-abc1234 -f compose.yml -f override.yml",
		"up -d --build --no-deps web",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildDockerPort(t *testing.T) {
	got := buildDockerPortCmd("myapp-abc1234-web", 8080)
	if !strings.Contains(got, "docker port myapp-abc1234-web 8080") {
		t.Errorf("got %s", got)
	}
}

func TestExtractHostPort(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"127.0.0.1:49231\n", 49231, true},
		{"0.0.0.0:49231\n127.0.0.1:49231\n", 49231, true},
		{"", 0, false},
		{"garbage", 0, false},
	}
	for _, c := range cases {
		got, err := extractHostPort(c.in)
		if c.ok && err != nil {
			t.Errorf("in=%q got err=%v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("in=%q expected error", c.in)
		}
		if got != c.want {
			t.Errorf("in=%q got %d, want %d", c.in, got, c.want)
		}
	}
}

func TestBuildScheduleDrainCmd(t *testing.T) {
	got := buildScheduleDrainCmd("/opt/conoha/myapp/old", "myapp-old", "myapp", "old", 30000)
	for _, want := range []string{
		"sleep 30",
		"docker compose -p myapp-old",
		"down",
		"nohup",
		// re-read guard: must cat CURRENT_SLOT and compare to this slot
		"cat '/opt/conoha/myapp/CURRENT_SLOT'",
		`[ "$cur" = 'old' ]`,
		"skip teardown",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildAccessoryUp(t *testing.T) {
	got := buildAccessoryUp("/opt/conoha/myapp/abc1234", "myapp-accessories", "compose.yml", "", "myapp", []string{"db", "redis"})
	for _, want := range []string{
		"touch '/opt/conoha/myapp/.env.server'",
		"docker compose --env-file '/opt/conoha/myapp/.env.server' -p myapp-accessories",
		"-f compose.yml",
		"up -d db redis",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "conoha-accessories-override.yml") {
		t.Errorf("override path leaked when overrideFile=\"\": %s", got)
	}
}

func TestBuildAccessoryUp_WithOverride(t *testing.T) {
	got := buildAccessoryUp("/opt/conoha/myapp/abc1234", "myapp-accessories", "compose.yml", "conoha-accessories-override.yml", "myapp", []string{"db", "dex"})
	for _, want := range []string{
		"--env-file '/opt/conoha/myapp/.env.server'",
		"-f compose.yml",
		"-f conoha-accessories-override.yml",
		"up -d db dex",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
