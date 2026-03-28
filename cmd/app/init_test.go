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
		{"docker compose up", "docker compose up -d --build"},
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

func TestGenerateInitScriptAppNameEmbedded(t *testing.T) {
	script := string(generateInitScript("test-app-123"))

	if !strings.Contains(script, `APP_NAME="test-app-123"`) {
		t.Error("script should set APP_NAME variable with the app name")
	}
}
