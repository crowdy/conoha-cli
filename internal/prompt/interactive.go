package prompt

import (
	"os"

	"golang.org/x/term"

	"github.com/crowdy/conoha-cli/internal/config"
)

// IsInteractive returns true if the current session supports interactive prompts.
// Returns false if stdin is not a TTY or if CONOHA_NO_INPUT is set.
func IsInteractive() bool {
	if config.IsNoInput() {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}
