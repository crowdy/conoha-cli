package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

// Password prompts for a password (no echo in future, plain for now).
func Password(label string) (string, error) {
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
