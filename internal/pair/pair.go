package pair

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func Resolve(beforeDir, afterDir, glob string) ([]model.Pair, error) {
	if glob == "" {
		glob = "*.cfg,*.conf,*.txt,*.log"
	}
	patterns := parsePatterns(glob)

	afterFiles := map[string]string{}
	err := filepath.WalkDir(afterDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		matched, err := matchesAny(filepath.Base(path), patterns)
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
		rel, err := filepath.Rel(afterDir, path)
		if err != nil {
			return err
		}
		afterFiles[rel] = path
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(afterFiles) == 0 {
		return nil, fmt.Errorf("no files matched glob %q in %s", glob, afterDir)
	}

	pairs := make([]model.Pair, 0, len(afterFiles))
	for rel, afterPath := range afterFiles {
		beforePath := filepath.Join(beforeDir, rel)
		if _, err := os.Stat(beforePath); err != nil {
			return nil, fmt.Errorf("before file not found for %s", rel)
		}

		beforeLines, err := readLines(beforePath)
		if err != nil {
			return nil, err
		}
		afterLines, err := readLines(afterPath)
		if err != nil {
			return nil, err
		}

		pairs = append(pairs, model.Pair{
			Name:   rel,
			Before: beforeLines,
			After:  afterLines,
		})
	}

	return pairs, nil
}

func parsePatterns(glob string) []string {
	raw := strings.Split(glob, ",")
	patterns := make([]string, 0, len(raw))
	for _, pattern := range raw {
		trimmed := strings.TrimSpace(pattern)
		if trimmed != "" {
			patterns = append(patterns, trimmed)
		}
	}
	if len(patterns) == 0 {
		return []string{"*.cfg", "*.conf", "*.txt", "*.log"}
	}
	return patterns
}

func matchesAny(name string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
