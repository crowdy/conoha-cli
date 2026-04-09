package app

import (
	"strings"
	"testing"
)

func TestGenerateInitScript(t *testing.T) {
	script := string(generateInitScript("myapp"))

	checks := []struct {
		name string
		want string
	}{
		{"shebang", "#!/bin/bash"},
		{"set strict", "set -euo pipefail"},
		{"docker install", "get.docker.com"},
		{"git install", "apt-get install -y -qq git"},
		{"app name var", `APP_NAME="myapp"`},
		{"repo dir", "/opt/conoha/${APP_NAME}.git"},
		{"work dir", "/opt/conoha/${APP_NAME}"},
		{"bare repo init", "git init --bare"},
		{"post-receive hook", "hooks/post-receive"},
		{"docker compose up", `docker compose -f "$COMPOSE_FILE" up -d --build`},
		{"deploy branch", `DEPLOY_BRANCH="main"`},
		{"branch check", `read -r oldrev newrev refname`},
		{"chmod", "chmod +x"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(script, c.want) {
				t.Errorf("script missing %q", c.want)
			}
		})
	}
}

func TestGenerateInitScriptComposeDetection(t *testing.T) {
	script := string(generateInitScript("myapp"))

	// Verify conoha-docker-compose.yml is checked first
	checks := []string{
		"conoha-docker-compose.yml",
		"conoha-docker-compose.yaml",
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"COMPOSE_FILE=",
		`docker compose -f "$COMPOSE_FILE"`,
	}
	for _, want := range checks {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q", want)
		}
	}

	// Verify priority order: conoha-docker-compose.yml appears before docker-compose.yml
	conohaIdx := strings.Index(script, "conoha-docker-compose.yml")
	dockerIdx := strings.Index(script, "docker-compose.yml")
	if conohaIdx > dockerIdx {
		t.Error("conoha-docker-compose.yml should be checked before docker-compose.yml")
	}
}

func TestGenerateInitScriptAppNameEmbedded(t *testing.T) {
	script := string(generateInitScript("test-app-123"))

	if !strings.Contains(script, `APP_NAME="test-app-123"`) {
		t.Error("script should set APP_NAME variable with the app name")
	}
}
