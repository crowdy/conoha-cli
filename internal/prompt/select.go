package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"

	"github.com/crowdy/conoha-cli/internal/config"
)

// SelectItem represents a selectable item.
type SelectItem struct {
	Label string
	Value string
}

// Select shows an interactive selection prompt and returns the selected Value.
// hint (optional) is shown in the error message when a TTY is unavailable,
// so the user knows which flag to use instead (e.g. "use --security-group <name>").
// Without a hint, the label is included to identify which selection failed.
func Select(label string, items []SelectItem, hint ...string) (string, error) {
	if config.IsNoInput() || !term.IsTerminal(int(os.Stdin.Fd())) {
		if len(hint) > 0 && hint[0] != "" {
			return "", fmt.Errorf("%s; interactive selection requires a TTY", hint[0])
		}
		return "", fmt.Errorf("interactive selection requires a TTY for %q; use flags to specify values", label)
	}

	searcher := func(input string, index int) bool {
		return strings.Contains(
			strings.ToLower(items[index].Label),
			strings.ToLower(input),
		)
	}

	prompt := promptui.Select{
		Label:    label,
		Items:    items,
		Size:     15,
		Searcher: searcher,
		Stdout:   os.Stderr,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }} (Ctrl+C to cancel):",
			Active:   "\u25b8 {{ .Label | cyan }}",
			Inactive: "  {{ .Label }}",
			Selected: "\u2713 {{ .Label | green }}",
		},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("selection cancelled: %w", err)
	}
	return items[idx].Value, nil
}
