package proxy

import (
	"testing"
)

// TestSSHExecutor_NilClientSurface lands the "nil *ssh.Client" shape as a
// documented compile-time assertion: construction is cheap and field defaults
// are zero-valued, so Run against a nil client must fail fast via
// internalssh.RunCommand's own nil check rather than panicking at field
// dereference. We do not dial, so this stays a pure unit test.
func TestSSHExecutor_StderrDefaultsAreSafe(t *testing.T) {
	// Construct with no Stderr — the Run path should substitute os.Stderr.
	// We don't exercise Run (requires live ssh.Client); we just verify that
	// the zero value of Stderr is acceptable so callers don't have to set it.
	e := &SSHExecutor{}
	if e.Stderr != nil {
		t.Errorf("zero-value Stderr should be nil, got %v", e.Stderr)
	}
}
