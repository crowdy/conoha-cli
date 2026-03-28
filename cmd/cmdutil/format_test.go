package cmdutil

import (
	"testing"

	"github.com/spf13/cobra"
)

func newCmdWithFormatFlag() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("format", "", "output format")
	return cmd
}

func TestGetFormat_Default(t *testing.T) {
	t.Setenv("CONOHA_CONFIG_DIR", t.TempDir())
	t.Setenv("CONOHA_FORMAT", "")
	cmd := newCmdWithFormatFlag()
	got := GetFormat(cmd)
	if got != "table" {
		t.Errorf("expected 'table', got %q", got)
	}
}

func TestGetFormat_EnvOverride(t *testing.T) {
	t.Setenv("CONOHA_FORMAT", "json")
	cmd := newCmdWithFormatFlag()
	got := GetFormat(cmd)
	if got != "json" {
		t.Errorf("expected 'json', got %q", got)
	}
}

func TestGetFormat_FlagOverride(t *testing.T) {
	t.Setenv("CONOHA_FORMAT", "json")
	cmd := newCmdWithFormatFlag()
	if err := cmd.Flags().Set("format", "yaml"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	got := GetFormat(cmd)
	if got != "yaml" {
		t.Errorf("expected 'yaml', got %q", got)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1024, "1 KB"},
		{1536, "1 KB"},
		{10240, "10 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
		{1099511627776, "1.0 TB"},
		{2199023255552, "2.0 TB"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
