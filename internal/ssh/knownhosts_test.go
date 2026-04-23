package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestHostKeyCallback_Insecure(t *testing.T) {
	cb, err := HostKeyCallback(true, false)
	if err != nil {
		t.Fatalf("Insecure path should never error, got %v", err)
	}
	// The insecure callback accepts any key without reading known_hosts.
	key := genKey(t)
	if err := cb("example.com:22", fakeTCPAddr(t, "1.2.3.4:22"), key); err != nil {
		t.Errorf("Insecure callback rejected key: %v", err)
	}
}

func TestHostKeyCallback_MismatchIsDistinctError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")
	t.Setenv("SSH_KNOWN_HOSTS", path)

	// Pre-populate known_hosts with a key for "example.com:22", then hand
	// the callback a different key for the same host.
	pinned := genKey(t)
	line := knownhosts.Line([]string{knownhosts.Normalize("example.com:22")}, pinned)
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		t.Fatalf("seeding known_hosts: %v", err)
	}

	cb, err := HostKeyCallback(false, true /* noInput — avoids TOFU prompt */)
	if err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}

	other := genKey(t)
	err = cb("example.com:22", fakeTCPAddr(t, "1.2.3.4:22"), other)
	if err == nil {
		t.Fatal("expected a mismatch error, got nil")
	}
	var mismatch *HostKeyMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected HostKeyMismatchError, got %T: %v", err, err)
	}
	if mismatch.Host != "example.com:22" {
		t.Errorf("mismatch.Host = %q, want example.com:22", mismatch.Host)
	}
	if !strings.Contains(mismatch.Error(), "ssh-keygen -R") {
		t.Errorf("mismatch error should suggest ssh-keygen -R, got: %s", mismatch.Error())
	}
}

func TestHostKeyCallback_UnknownHost_NoInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")
	t.Setenv("SSH_KNOWN_HOSTS", path)

	cb, err := HostKeyCallback(false, true /* noInput */)
	if err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}

	key := genKey(t)
	err = cb("new-host:22", fakeTCPAddr(t, "1.2.3.4:22"), key)
	if err == nil {
		t.Fatal("expected refusal in no-input mode, got nil")
	}
	if !strings.Contains(err.Error(), "not in") || !strings.Contains(err.Error(), "--insecure") {
		t.Errorf("expected helpful no-input error message, got: %v", err)
	}
}

func TestHostKeyCallback_CreatesMissingKnownHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ssh-sub", "known_hosts")
	t.Setenv("SSH_KNOWN_HOSTS", path)

	if _, err := HostKeyCallback(false, true); err != nil {
		t.Fatalf("HostKeyCallback should auto-create the file: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to be created, got %v", path, err)
	}
}

// --- helpers ---

func genKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("ssh.NewPublicKey: %v", err)
	}
	return k
}

func fakeTCPAddr(t *testing.T, s string) net.Addr {
	t.Helper()
	a, err := net.ResolveTCPAddr("tcp", s)
	if err != nil {
		t.Fatalf("ResolveTCPAddr: %v", err)
	}
	return a
}
