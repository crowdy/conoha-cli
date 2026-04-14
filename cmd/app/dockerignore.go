package app

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var defaultExcludes = []string{".git"}

// loadIgnorePatterns reads .dockerignore files from dir and all subdirectories.
// Patterns from subdirectory .dockerignore files are prefixed with the relative
// directory path so they apply only within that subtree.
func loadIgnorePatterns(dir string) ([]string, error) {
	patterns := append([]string{}, defaultExcludes...)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != ".dockerignore" {
			return nil
		}

		prefix, err := filepath.Rel(dir, filepath.Dir(path))
		if err != nil {
			return err
		}
		sub, err := readIgnoreFile(path, prefix)
		if err != nil {
			return err
		}
		patterns = append(patterns, sub...)
		return nil
	})

	return patterns, err
}

func readIgnoreFile(path, prefix string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		line = strings.TrimRight(line, "/")
		if prefix != "." {
			line = prefix + "/" + line
		}
		result = append(result, line)
	}
	return result, scanner.Err()
}

func shouldExclude(path string, patterns []string) bool {
	base := filepath.Base(path)
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if matched, _ := filepath.Match(p, base); matched {
			return true
		}
	}
	return false
}
