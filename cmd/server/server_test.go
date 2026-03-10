package server

import "testing"

func TestFormatMB(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{512, "512M"},
		{1024, "1G"},
		{2048, "2G"},
		{3072, "3G"},
		{1536, "1536M"},
	}
	for _, tt := range tests {
		got := formatMB(tt.input)
		if got != tt.want {
			t.Errorf("formatMB(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
