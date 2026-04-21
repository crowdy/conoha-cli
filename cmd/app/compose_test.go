package app

import (
	"strings"
	"testing"
)

func TestComposeProjectEnumPipeline_UsesLabelSource(t *testing.T) {
	got := composeProjectEnumPipeline("myapp")
	if !strings.Contains(got, `docker ps -a --format '{{.Label "com.docker.compose.project"}}'`) {
		t.Errorf("pipeline should read the compose project label via docker ps; got: %s", got)
	}
	if strings.Contains(got, `docker compose ls`) {
		t.Errorf("pipeline must not fall back to `docker compose ls` (unsupported --format on Compose v5, #114); got: %s", got)
	}
	if strings.Contains(got, "{{.Name}}") {
		t.Errorf("pipeline must not use the legacy Go-template format; got: %s", got)
	}
}

func TestComposeProjectEnumPipeline_FiltersByAppAndSlot(t *testing.T) {
	got := composeProjectEnumPipeline("myapp")
	if !strings.Contains(got, `grep -E "^myapp(-|$)"`) {
		t.Errorf("pipeline should filter project names with the app-or-slot regex; got: %s", got)
	}
}

func TestComposeProjectEnumPipeline_TerminatesWithOrTrue(t *testing.T) {
	got := composeProjectEnumPipeline("myapp")
	// `grep` exits 1 when no lines match; the pipeline must absorb that
	// under `set -euo pipefail` so empty enumerations don't abort callers.
	if !strings.HasSuffix(strings.TrimSpace(got), "|| true") {
		t.Errorf("pipeline should end in `|| true` so no-match is a soft success; got: %s", got)
	}
}
