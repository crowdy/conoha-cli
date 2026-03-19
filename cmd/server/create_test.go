package server

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmd(flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("user-data", "", "")
	cmd.Flags().String("user-data-raw", "", "")
	cmd.Flags().String("user-data-url", "", "")
	for k, v := range flags {
		_ = cmd.Flags().Set(k, v)
	}
	return cmd
}

func TestResolveUserData_None(t *testing.T) {
	cmd := newTestCmd(nil)
	got, err := resolveUserData(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestResolveUserData_Raw(t *testing.T) {
	cmd := newTestCmd(map[string]string{"user-data-raw": "#!/bin/bash\necho hello"})
	got, err := resolveUserData(cmd)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "#!/bin/bash\necho hello" {
		t.Errorf("decoded = %q", string(decoded))
	}
}

func TestResolveUserData_URL(t *testing.T) {
	cmd := newTestCmd(map[string]string{"user-data-url": "https://example.com/setup.sh"})
	got, err := resolveUserData(cmd)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	want := "#include\nhttps://example.com/setup.sh\n"
	if string(decoded) != want {
		t.Errorf("decoded = %q, want %q", string(decoded), want)
	}
}

func TestResolveUserData_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "startup.sh")
	content := "#!/bin/bash\napt update"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd(map[string]string{"user-data": path})
	got, err := resolveUserData(cmd)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != content {
		t.Errorf("decoded = %q, want %q", string(decoded), content)
	}
}

func TestResolveUserData_FileNotFound(t *testing.T) {
	cmd := newTestCmd(map[string]string{"user-data": "/nonexistent/file.sh"})
	_, err := resolveUserData(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveUserData_TooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.sh")
	data := strings.Repeat("x", userDataMaxSize+1)
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd(map[string]string{"user-data": path})
	_, err := resolveUserData(cmd)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %v", err)
	}
}

func TestResolveUserData_MutualExclusion(t *testing.T) {
	cmd := newTestCmd(map[string]string{
		"user-data-raw": "echo hi",
		"user-data-url": "https://example.com/x.sh",
	})
	_, err := resolveUserData(cmd)
	if err == nil {
		t.Fatal("expected error for multiple flags")
	}
	if !strings.Contains(err.Error(), "only one") {
		t.Errorf("expected 'only one' error, got: %v", err)
	}
}
