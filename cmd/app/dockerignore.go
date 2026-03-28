package app

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var defaultExcludes = []string{".git"}

func loadIgnorePatterns(dir string) ([]string, error) {
	patterns := append([]string{}, defaultExcludes...)

	f, err := os.Open(filepath.Join(dir, ".dockerignore"))
	if err != nil {
		if os.IsNotExist(err) {
			return patterns, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "!") {
			continue
		}
		line = strings.TrimRight(line, "/")
		patterns = append(patterns, line)
	}
	return patterns, scanner.Err()
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
