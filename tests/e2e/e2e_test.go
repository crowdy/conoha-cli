//go:build e2e

// Package e2e drives the Phase 1 end-to-end harness (spec
// docs/superpowers/specs/2026-04-23-e2e-tests-design.md §8 Phase 1).
// Excluded from `go test ./...` by the `e2e` build tag; run explicitly
// with `go test -tags e2e ./tests/e2e/`.
package e2e

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPhase1Harness(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(thisFile), "run.sh")

	cmd := exec.Command("bash", script)
	cmd.Stdout = testWriter{t}
	cmd.Stderr = testWriter{t}
	if err := cmd.Run(); err != nil {
		t.Fatalf("run.sh failed: %v", err)
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", p)
	return len(p), nil
}
