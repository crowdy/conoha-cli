package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCreateTarGz(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte("version: '3'"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}

	patterns := []string{".git", "*.log"}
	var buf bytes.Buffer
	if err := createTarGz(dir, patterns, &buf); err != nil {
		t.Fatal(err)
	}

	files := extractTarNames(t, &buf)
	sort.Strings(files)

	want := []string{"app.go", "compose.yml"}
	if len(files) != len(want) {
		t.Fatalf("got files %v, want %v", files, want)
	}
	for i, f := range files {
		if f != want[i] {
			t.Errorf("file[%d]: got %q, want %q", i, f, want[i])
		}
	}
}

func TestCreateTarGz_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := createTarGz(dir, nil, &buf); err != nil {
		t.Fatal(err)
	}

	files := extractTarNames(t, &buf)
	if len(files) != 0 {
		t.Errorf("expected empty archive, got %v", files)
	}
}

func extractTarNames(t *testing.T, data *bytes.Buffer) []string {
	t.Helper()
	gr, err := gzip.NewReader(data)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := gr.Close(); err != nil {
			t.Errorf("gzip close: %v", err)
		}
	}()

	tr := tar.NewReader(gr)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !hdr.FileInfo().IsDir() {
			names = append(names, hdr.Name)
		}
	}
	return names
}
