package app

import (
	"strings"
	"testing"
)

func TestDestroyCmd_HasYesFlag(t *testing.T) {
	f := destroyCmd.Flags().Lookup("yes")
	if f == nil {
		t.Fatal("destroy command should have --yes flag")
	}
	if f.DefValue != "false" {
		t.Errorf("--yes default should be false, got %s", f.DefValue)
	}
}

func TestDestroyCmd_HasModeFlags(t *testing.T) {
	if destroyCmd.Flags().Lookup("proxy") == nil {
		t.Error("destroy should have --proxy flag")
	}
	if destroyCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("destroy should have --no-proxy flag")
	}
}

// Docker Compose v5 dropped Go-template --format support for `docker compose ls`
// (only `table` and `json` are accepted). Emitting `--format '{{.Name}}'` silently
// fails on v5 hosts so the enumeration loop iterates over nothing and containers
// leak. Regression: issue #114.
func TestGenerateDestroyScript_DoesNotUseLegacyGoTemplateFormat(t *testing.T) {
	script := string(generateDestroyScript("myapp"))
	if strings.Contains(script, "{{.Name}}") {
		t.Errorf("destroy script must not use docker compose ls --format '{{.Name}}' (unsupported on Compose v5); script:\n%s", script)
	}
}

// The enumeration must survive across docker compose versions. Labels on
// containers (com.docker.compose.project) read via `docker ps -a` are the
// stable source of truth; `docker inspect` or other approaches would not
// satisfy the #114 fix (addresses review NP1).
func TestGenerateDestroyScript_EnumeratesViaComposeProjectLabel(t *testing.T) {
	script := string(generateDestroyScript("myapp"))
	if !strings.Contains(script, "com.docker.compose.project") {
		t.Errorf("destroy script should enumerate compose projects via the com.docker.compose.project label; script:\n%s", script)
	}
	if !strings.Contains(script, `docker ps -a --format '{{.Label "com.docker.compose.project"}}'`) {
		t.Errorf("destroy script should read labels via `docker ps -a --format`; script:\n%s", script)
	}
}

func TestGenerateDestroyScript_InterpolatesAppName(t *testing.T) {
	script := string(generateDestroyScript("myapp"))
	if !strings.Contains(script, `APP_NAME="myapp"`) {
		t.Errorf(`destroy script should set APP_NAME="myapp"; script:\n%s`, script)
	}
	if !strings.Contains(script, `APP_DIR="/opt/conoha/${APP_NAME}"`) {
		t.Errorf(`destroy script should derive APP_DIR from APP_NAME; script:\n%s`, script)
	}
}

// The project-name match must cover exactly <app> (no-proxy) and <app>-<slot>
// (proxy). It must not match <app>foo or other unrelated projects.
func TestGenerateDestroyScript_MatchesAppAndSlotProjectsOnly(t *testing.T) {
	script := string(generateDestroyScript("myapp"))
	if !strings.Contains(script, `"^${APP_NAME}(-|$)"`) {
		t.Errorf(`destroy script should filter projects with "^${APP_NAME}(-|$)"; script:\n%s`, script)
	}
}
