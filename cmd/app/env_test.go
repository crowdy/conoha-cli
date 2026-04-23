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
		// Data-loss guard must appear; the script must abort on legacy-only.
		`exit 2`,
		`'conoha app env migrate`,
		// Credentials-bearing file: 0600 enforced.
		`chmod 600 "$ENV_FILE"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in script:\n%s", want, s)
		}
	}
	// Write target must be the new path only. Legacy appears in the guard
	// as a source-side check, never on the left side of a write-redirect.
	for _, forbidden := range []string{
		`> "/opt/conoha/myapp.env.server"`,
		`>> "/opt/conoha/myapp.env.server"`,
		`mv "$ENV_FILE.tmp" "/opt/conoha/myapp.env.server"`,
	} {
		if strings.Contains(s, forbidden) {
			t.Errorf("set script must not write to legacy path (%q):\n%s", forbidden, s)
		}
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
		// Data-loss guard symmetric with set.
		`exit 2`,
		// Mode enforced after the final mv.
		`chmod 600 "$ENV_FILE"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in unset script:\n%s", want, s)
		}
	}
}

func TestLegacyOnlyGuardRejects(t *testing.T) {
	s := legacyOnlyGuardScript("myapp")
	for _, want := range []string{
		`NEW_ENV="/opt/conoha/myapp/.env.server"`,
		`LEGACY_ENV="/opt/conoha/myapp.env.server"`,
		`if [ ! -f "$NEW_ENV" ] && [ -f "$LEGACY_ENV" ]`,
		`Writing here would silently hide legacy values`,
		`exit 2`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in guard script:\n%s", want, s)
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
		// `mv` preserves the source mode; enforce 0600 post-move since the
		// legacy path may have been 0644 on old servers.
		`chmod 600 "$NEW_ENV"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in migrate script:\n%s", want, s)
		}
	}
}
