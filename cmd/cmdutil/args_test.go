package cmdutil

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExactArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "show <id>"}
	cmd.SetArgs([]string{})

	tests := []struct {
		name    string
		n       int
		args    []string
		wantErr bool
		contain string
	}{
		{"ok", 1, []string{"abc"}, false, ""},
		{"too few", 1, []string{}, true, "show <id>"},
		{"too many", 1, []string{"a", "b"}, true, "show <id>"},
		{"two args ok", 2, []string{"a", "b"}, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExactArgs(tt.n)(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExactArgs(%d)(%v) error = %v, wantErr %v", tt.n, tt.args, err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.contain) {
				t.Errorf("error %q should contain %q", err.Error(), tt.contain)
			}
		})
	}
}
