package server

import (
	"strings"
	"testing"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func TestParseEnvFlags(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantKey string
		wantVal string
		wantErr bool
	}{
		{"valid", "FOO=bar", "FOO", "bar", false},
		{"empty value", "FOO=", "FOO", "", false},
		{"value with equals", "FOO=a=b", "FOO", "a=b", false},
		{"missing equals", "FOO", "", "", true},
		{"invalid key", "1BAD=val", "", "", true},
		{"shell injection key", "FOO;rm=val", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, v, ok := strings.Cut(tt.input, "=")
			if !ok {
				if !tt.wantErr {
					t.Fatalf("unexpected parse failure for %q", tt.input)
				}
				return
			}
			if err := internalssh.ValidateEnvKey(k); err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("expected error, got none")
			}
			if k != tt.wantKey {
				t.Errorf("key: got %q, want %q", k, tt.wantKey)
			}
			if v != tt.wantVal {
				t.Errorf("val: got %q, want %q", v, tt.wantVal)
			}
		})
	}
}
