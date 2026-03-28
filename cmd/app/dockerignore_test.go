package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnorePatterns_NoFile(t *testing.T) {
	dir := t.TempDir()
	patterns, err := loadIgnorePatterns(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 1 || patterns[0] != ".git" {
		t.Errorf("expected [.git], got %v", patterns)
	}
}

func TestLoadIgnorePatterns_WithFile(t *testing.T) {
	dir := t.TempDir()
	content := "node_modules\n# comment\n\n*.log\nlogs/\n!important.log\n"
	if err := os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := loadIgnorePatterns(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{".git", "node_modules", "*.log", "logs"}
	if len(patterns) != len(want) {
		t.Fatalf("got %v, want %v", patterns, want)
	}
	for i, p := range patterns {
		if p != want[i] {
			t.Errorf("pattern[%d]: got %q, want %q", i, p, want[i])
		}
	}
}

func TestShouldExclude(t *testing.T) {
	patterns := []string{".git", "node_modules", "*.log"}

	tests := []struct {
		path string
		want bool
	}{
		{".git", true},
		{"node_modules", true},
		{"src/app.go", false},
		{"error.log", true},
		{"src/error.log", true},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := shouldExclude(tt.path, patterns); got != tt.want {
				t.Errorf("shouldExclude(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
