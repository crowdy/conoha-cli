package app

import (
	"strings"
	"testing"
)

func TestGenerateEnvSetScript(t *testing.T) {
	script := generateEnvSetScript("myapp", map[string]string{
		"DB_HOST": "localhost",
		"DB_PORT": "5432",
	})
	s := string(script)

	if !strings.Contains(s, `ENV_FILE="/opt/conoha/myapp.env.server"`) {
		t.Error("missing ENV_FILE path")
	}
	if !strings.Contains(s, "touch") {
		t.Error("missing touch command")
	}
	if !strings.Contains(s, "DB_HOST=localhost") {
		t.Error("missing DB_HOST=localhost")
	}
	if !strings.Contains(s, "DB_PORT=5432") {
		t.Error("missing DB_PORT=5432")
	}
}

func TestGenerateEnvUnsetScript(t *testing.T) {
	script := generateEnvUnsetScript("myapp", []string{"DB_HOST", "DB_PORT"})
	s := string(script)

	if !strings.Contains(s, `ENV_FILE="/opt/conoha/myapp.env.server"`) {
		t.Error("missing ENV_FILE path")
	}
	if !strings.Contains(s, `grep -v "^DB_HOST="`) {
		t.Error("missing grep for DB_HOST")
	}
	if !strings.Contains(s, `grep -v "^DB_PORT="`) {
		t.Error("missing grep for DB_PORT")
	}
}

func TestGenerateEnvGetCommand(t *testing.T) {
	cmd := generateEnvGetCommand("myapp", "DB_HOST")
	if !strings.Contains(cmd, `grep "^DB_HOST="`) {
		t.Error("missing grep for DB_HOST")
	}
	if !strings.Contains(cmd, "cut -d= -f2-") {
		t.Error("missing cut command")
	}
}

func TestGenerateEnvListCommand(t *testing.T) {
	cmd := generateEnvListCommand("myapp")
	if !strings.Contains(cmd, "/opt/conoha/myapp.env.server") {
		t.Error("missing env file path")
	}
}
