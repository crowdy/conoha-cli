package ssh

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/crowdy/conoha-cli/internal/model"
)

// ServerIP extracts the best IPv4 address from server addresses.
// Prefers floating IPv4 over fixed IPv4, falls back to any IPv4.
func ServerIP(s *model.Server) (string, error) {
	var fixedIP, floatingIP, anyIP string

	for _, addrs := range s.Addresses {
		for _, a := range addrs {
			if a.Version != 4 {
				continue
			}
			switch a.Type {
			case "floating":
				floatingIP = a.Addr
			case "fixed":
				if fixedIP == "" {
					fixedIP = a.Addr
				}
			default:
				if anyIP == "" {
					anyIP = a.Addr
				}
			}
		}
	}

	if floatingIP != "" {
		return floatingIP, nil
	}
	if fixedIP != "" {
		return fixedIP, nil
	}
	if anyIP != "" {
		return anyIP, nil
	}
	return "", fmt.Errorf("no IPv4 address found for server %s (%s)", s.Name, s.ID)
}

// ResolveKeyPath returns the SSH private key path for a server's key name.
// Looks for ~/.ssh/conoha_<keyName>. Returns empty string if not found.
func ResolveKeyPath(keyName string) string {
	if keyName == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	keyPath := filepath.Join(home, ".ssh", "conoha_"+keyName)
	if _, err := os.Stat(keyPath); err != nil {
		return ""
	}
	return keyPath
}
