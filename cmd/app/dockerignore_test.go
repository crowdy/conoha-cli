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

func TestLoadIgnorePatterns_SubdirDockerignore(t *testing.T) {
	dir := t.TempDir()
	// Root .dockerignore does NOT list node_modules
	if err := os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// frontend subdirectory has its own .dockerignore
	if err := os.MkdirAll(filepath.Join(dir, "frontend"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "frontend", ".dockerignore"), []byte("node_modules\n.next\n"), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := loadIgnorePatterns(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !shouldExclude("frontend/node_modules", patterns) {
		t.Error("frontend/node_modules should be excluded by frontend/.dockerignore")
	}
	if !shouldExclude("frontend/.next", patterns) {
		t.Error("frontend/.next should be excluded by frontend/.dockerignore")
	}
	// backend/node_modules must NOT be excluded (no pattern covers it)
	if shouldExclude("backend/node_modules", patterns) {
		t.Error("backend/node_modules should NOT be excluded")
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
