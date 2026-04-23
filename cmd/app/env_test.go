package app

import (
	"strings"
	"testing"
)

func TestEnvFilePaths(t *testing.T) {
	if got := envFilePath("myapp"); got != "/opt/conoha/myapp/.env.server" {
		t.Errorf("envFilePath = %q, want /opt/conoha/myapp/.env.server", got)
	}
	if got := legacyEnvFilePath("myapp"); got != "/opt/conoha/myapp.env.server" {
		t.Errorf("legacyEnvFilePath = %q, want /opt/conoha/myapp.env.server", got)
	}
}

func TestGenerateEnvSetScript(t *testing.T) {
	script := generateEnvSetScript("myapp", map[string]string{
		"DB_HOST": "localhost",
		"DB_PORT": "5432",
	})
	s := string(script)

	for _, want := range []string{
		`"/opt/conoha/myapp/.env.server"`, // new canonical path
		"mkdir -p '/opt/conoha/myapp'",
		"touch",
		"DB_HOST=localhost",
		"DB_PORT=5432",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in script:\n%s", want, s)
		}
	}
	// Set writes to new path only; legacy path must not appear in a write
	// context (it still shows up on the read side for backward compat).
	if strings.Contains(s, legacyEnvFilePath("myapp")) {
		t.Errorf("set script should not reference the legacy path: %s", s)
	}
}

func TestGenerateEnvUnsetScript(t *testing.T) {
	script := generateEnvUnsetScript("myapp", []string{"DB_HOST", "DB_PORT"})
	s := string(script)

	for _, want := range []string{
		"NEW_ENV=\"/opt/conoha/myapp/.env.server\"",
		"LEGACY_ENV=\"/opt/conoha/myapp.env.server\"",
		`grep -v "^DB_HOST="`,
		`grep -v "^DB_PORT="`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in unset script:\n%s", want, s)
		}
	}
}

func TestGenerateEnvGetCommand(t *testing.T) {
	cmd := generateEnvGetCommand("myapp", "DB_HOST")
	for _, want := range []string{
		`grep "^DB_HOST="`,
		"cut -d= -f2-",
		"/opt/conoha/myapp/.env.server",
		"/opt/conoha/myapp.env.server", // legacy referenced in fallback chain
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("missing %q in get command:\n%s", want, cmd)
		}
	}
}

func TestGenerateEnvListCommand(t *testing.T) {
	cmd := generateEnvListCommand("myapp")
	for _, want := range []string{
		"/opt/conoha/myapp/.env.server",
		"/opt/conoha/myapp.env.server",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("missing %q in list command:\n%s", want, cmd)
		}
	}
}

func TestGenerateEnvMigrateScript(t *testing.T) {
	s := string(generateEnvMigrateScript("myapp"))
	for _, want := range []string{
		`NEW_ENV="/opt/conoha/myapp/.env.server"`,
		`LEGACY_ENV="/opt/conoha/myapp.env.server"`,
		"Nothing to migrate",
		"Both",
		`mv "$LEGACY_ENV" "$NEW_ENV"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in migrate script:\n%s", want, s)
		}
	}
}
