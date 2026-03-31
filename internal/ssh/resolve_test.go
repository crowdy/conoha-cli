package ssh

import (
	"testing"

	"github.com/crowdy/conoha-cli/internal/model"
)

func TestServerIP(t *testing.T) {
	tests := []struct {
		name    string
		server  *model.Server
		want    string
		wantErr bool
	}{
		{
			name: "floating preferred over fixed",
			server: &model.Server{
				Name: "test",
				Addresses: map[string][]model.Address{
					"net1": {
						{Addr: "10.0.0.1", Version: 4, Type: "fixed"},
						{Addr: "203.0.113.1", Version: 4, Type: "floating"},
					},
				},
			},
			want: "203.0.113.1",
		},
		{
			name: "fixed only",
			server: &model.Server{
				Name: "test",
				Addresses: map[string][]model.Address{
					"net1": {
						{Addr: "10.0.0.1", Version: 4, Type: "fixed"},
					},
				},
			},
			want: "10.0.0.1",
		},
		{
			name: "ipv6 only returns error",
			server: &model.Server{
				Name: "test",
				Addresses: map[string][]model.Address{
					"net1": {
						{Addr: "2001:db8::1", Version: 6, Type: "fixed"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no addresses",
			server: &model.Server{
				Name:      "test",
				Addresses: map[string][]model.Address{},
			},
			wantErr: true,
		},
		{
			name: "unknown type fallback",
			server: &model.Server{
				Name: "test",
				ID:   "abc-123",
				Addresses: map[string][]model.Address{
					"net1": {
						{Addr: "10.0.0.1", Version: 4, Type: ""},
					},
				},
			},
			want: "10.0.0.1",
		},
		{
			name: "multiple networks",
			server: &model.Server{
				Name: "test",
				Addresses: map[string][]model.Address{
					"net1": {
						{Addr: "10.0.0.1", Version: 4, Type: "fixed"},
					},
					"net2": {
						{Addr: "192.168.0.1", Version: 4, Type: "floating"},
					},
				},
			},
			want: "192.168.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ServerIP(tt.server)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveKeyPath(t *testing.T) {
	if got := ResolveKeyPath(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	if got := ResolveKeyPath("nonexistent-key-12345"); got != "" {
		t.Errorf("expected empty for non-existent key, got %q", got)
	}
}
