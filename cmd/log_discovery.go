package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leo/leo-cli/internal/config"
)

var conventionalLogRoots = []string{
	"runtime/logs",
	"logs",
	"log",
	"var/log",
	"storage/logs",
}

var skippedLogDiscoveryDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
	"target":       {},
	".venv":        {},
	"venv":         {},
	"__pycache__":  {},
}

func discoverLogRoots(root string) ([]string, []string) {
	root = filepath.Clean(root)
	roots := make([]string, 0, len(conventionalLogRoots))
	warnings := make([]string, 0)
	for _, relative := range conventionalLogRoots {
		path := filepath.Join(root, relative)
		info, err := os.Stat(path)
		if err == nil {
			if info.IsDir() {
				roots = append(roots, path)
			} else {
				warnings = append(warnings, fmt.Sprintf("discover log directory %q: not a directory", path))
			}
			continue
		}
		if !os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf("discover log directory %q: %v", path, err))
		}
	}
	if len(roots) > 0 {
		return uniqueSortedPaths(roots), sortedStrings(warnings)
	}

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, fmt.Sprintf("discover log directory %q: %v", path, walkErr))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root || !entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("discover log directory %q: %v", path, err))
			return filepath.SkipDir
		}
		depth := len(strings.Split(relative, string(filepath.Separator)))
		if depth > 4 {
			return filepath.SkipDir
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		if _, skip := skippedLogDiscoveryDirs[strings.ToLower(name)]; skip {
			return filepath.SkipDir
		}
		if strings.EqualFold(name, "log") || strings.EqualFold(name, "logs") {
			roots = append(roots, filepath.Clean(path))
			return filepath.SkipDir
		}
		return nil
	})
	if walkErr != nil {
		warnings = append(warnings, fmt.Sprintf("discover log directories under %q: %v", root, walkErr))
	}
	return uniqueSortedPaths(roots), sortedStrings(warnings)
}

func expandExplicitLogRoots(cwd string, values []string) ([]string, error) {
	cleanCWD, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve current directory: %w", err)
	}
	roots := make([]string, 0, len(values))
	for _, value := range values {
		expanded := os.ExpandEnv(strings.TrimSpace(value))
		if expanded == "" {
			return nil, fmt.Errorf("log directory path is empty")
		}
		if expanded == "~" || strings.HasPrefix(expanded, "~/") {
			expanded, err = config.ExpandPath(expanded)
			if err != nil {
				return nil, fmt.Errorf("expand log directory %q: %w", value, err)
			}
		} else if !filepath.IsAbs(expanded) {
			expanded = filepath.Join(cleanCWD, expanded)
		}
		expanded, err = filepath.Abs(expanded)
		if err != nil {
			return nil, fmt.Errorf("resolve log directory %q: %w", value, err)
		}
		roots = append(roots, filepath.Clean(expanded))
	}
	return uniqueSortedPaths(roots), nil
}

func uniqueSortedPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		cleaned := filepath.Clean(path)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	sort.Strings(result)
	return result
}

func sortedStrings(values []string) []string {
	sort.Strings(values)
	return values
}
