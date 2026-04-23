package ssh

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// HostKeyCallback returns an ssh.HostKeyCallback that verifies the remote
// host key against ~/.ssh/known_hosts. On first connect to an unknown host
// it prompts the operator to accept and pin the key (TOFU). When noInput
// is true (CONOHA_NO_INPUT) or stdin is not a TTY, the connection fails
// rather than silently trusting.
//
// insecure=true returns the legacy InsecureIgnoreHostKey callback for lab
// and throwaway-VPS use; documented as the explicit opt-out for operators
// who knowingly want the old v0.1.x behavior back.
func HostKeyCallback(insecure, noInput bool) (ssh.HostKeyCallback, error) {
	if insecure {
		return ssh.InsecureIgnoreHostKey(), nil //nolint:gosec // user-requested via --insecure
	}

	path, err := knownHostsPath()
	if err != nil {
		return nil, err
	}

	// knownhosts.New rejects a missing file. Create an empty one so the
	// TOFU prompt path can append to it on first use.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("creating %s dir: %w", filepath.Dir(path), err)
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, fmt.Errorf("creating %s: %w", path, err)
		}
		_ = f.Close()
	}

	strict, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := strict(hostname, remote, key); err == nil {
			return nil
		} else if kkErr, ok := err.(*knownhosts.KeyError); ok {
			if len(kkErr.Want) > 0 {
				// Key mismatch — never auto-accept; this is a MITM signal.
				return &HostKeyMismatchError{Host: hostname, Path: path, Err: kkErr}
			}
			// Unknown host: TOFU prompt — only when stdin is genuinely
			// interactive. A non-TTY stdin (CI, build script piping a
			// heredoc, wrapper without --no-input) would otherwise let
			// `yes\n` from an untrusted source silently trust the host.
			if noInput || !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("host %s not in %s and stdin is not interactive (no-input mode or non-TTY) — refusing to trust unknown host. Add manually with ssh-keyscan or use --insecure", hostname, path)
			}
			return promptAndPin(path, hostname, remote, key)
		} else {
			return err
		}
	}, nil
}

// HostKeyMismatchError is returned when the server presents a host key that
// disagrees with the one pinned in known_hosts. Deliberately distinct from a
// plain error so callers can print MITM-specific guidance.
type HostKeyMismatchError struct {
	Host string
	Path string
	Err  error
}

func (e *HostKeyMismatchError) Error() string {
	return fmt.Sprintf(
		"host key for %s has changed! This is either the server was rebuilt or a man-in-the-middle attack.\n"+
			"  Pinned in: %s\n"+
			"  Underlying: %v\n"+
			"If you just rebuilt the VPS, run: ssh-keygen -R %s  (removes the old pin, next connect re-pins).",
		e.Host, e.Path, e.Err, e.Host)
}

func (e *HostKeyMismatchError) Unwrap() error { return e.Err }

// promptAndPin asks the user to accept the unknown key, then appends it to
// known_hosts in the canonical OpenSSH format.
func promptAndPin(path, hostname string, remote net.Addr, key ssh.PublicKey) error {
	fp := ssh.FingerprintSHA256(key)
	fmt.Fprintf(os.Stderr, "\nThe authenticity of host %q can't be established.\n", hostname)
	fmt.Fprintf(os.Stderr, "%s key fingerprint is %s.\n", key.Type(), fp)
	fmt.Fprint(os.Stderr, "Are you sure you want to continue connecting (yes/no)? ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading prompt answer: %w", err)
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	if answer != "yes" && answer != "y" {
		return fmt.Errorf("host %s key rejected by user", hostname)
	}

	// Canonical line: "<addresses> <keytype> <base64 key>"
	// knownhosts.Normalize returns host[:port] → host when port is 22.
	addr := knownhosts.Normalize(hostname)
	// Also include the numeric address so that later SSH sessions by IP
	// (common in this CLI — we connect to IPs, not names) also match.
	addrs := []string{addr}
	if _, ok := remote.(*net.TCPAddr); ok {
		if na := knownhosts.Normalize(remote.String()); na != addr {
			addrs = append(addrs, na)
		}
	}
	line = knownhosts.Line(addrs, key)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening %s for append: %w", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	fmt.Fprintf(os.Stderr, "Warning: Permanently added %q (%s) to the list of known hosts.\n", hostname, key.Type())
	return nil
}

// knownHostsPath returns the path to the user's known_hosts file.
// Honors SSH_KNOWN_HOSTS override for tests and bespoke setups.
func knownHostsPath() (string, error) {
	if p := os.Getenv("SSH_KNOWN_HOSTS"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving $HOME: %w", err)
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}
