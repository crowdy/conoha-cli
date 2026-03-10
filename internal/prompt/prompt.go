package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/crowdy/conoha-cli/internal/config"
)

// String prompts the user for a string input.
func String(label string) (string, error) {
	if config.IsNoInput() {
		return "", fmt.Errorf("input required but --no-input is set")
	}
	fmt.Fprintf(os.Stderr, "%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

// Password prompts for a password with masked input (shows *).
// Reads in raw terminal mode to support paste on WSL2.
func Password(label string) (string, error) {
	if config.IsNoInput() {
		return "", fmt.Errorf("input required but --no-input is set")
	}
	fmt.Fprintf(os.Stderr, "%s: ", label)

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback to term.ReadPassword if raw mode fails
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	var password []byte
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}
		switch buf[0] {
		case '\r', '\n':
			fmt.Fprint(os.Stderr, "\r\n")
			return strings.TrimSpace(string(password)), nil
		case 3: // Ctrl+C
			fmt.Fprint(os.Stderr, "\r\n")
			return "", fmt.Errorf("interrupted")
		case 127, 8: // Backspace, Ctrl+H
			if len(password) > 0 {
				password = password[:len(password)-1]
				fmt.Fprint(os.Stderr, "\b \b")
			}
		default:
			password = append(password, buf[0])
			fmt.Fprint(os.Stderr, "*")
		}
	}
	fmt.Fprint(os.Stderr, "\r\n")
	return strings.TrimSpace(string(password)), nil
}

// Confirm asks for yes/no confirmation.
func Confirm(label string) (bool, error) {
	if config.IsNoInput() {
		return false, fmt.Errorf("confirmation required but --no-input is set")
	}
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", label)
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "y" || s == "yes", nil
}
