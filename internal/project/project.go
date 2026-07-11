package project

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leo/leo-cli/internal/config"
)

type Selection struct {
	Name   string
	Root   string
	Config config.ProjectConfig
}

func Resolve(cwd, requested string, projects map[string]config.ProjectConfig) (Selection, error) {
	cleanCWD, err := filepath.Abs(cwd)
	if err != nil {
		return Selection{}, fmt.Errorf("resolve working directory: %w", err)
	}
	cleanCWD = filepath.Clean(cleanCWD)

	if requested != "" {
		projectConfig, ok := projects[requested]
		if !ok {
			return Selection{}, fmt.Errorf("unknown project %q; configured projects: %s", requested, projectNames(projects))
		}
		if root, ok := matchingAncestor(cleanCWD, projectMatch(requested, projectConfig)); ok {
			return Selection{Name: requested, Root: root, Config: projectConfig}, nil
		}
		return Selection{}, fmt.Errorf("working directory %q is not inside project %q", cleanCWD, requested)
	}

	names := sortedProjectNames(projects)
	for dir := cleanCWD; ; dir = filepath.Dir(dir) {
		matches := make([]string, 0, 1)
		for _, name := range names {
			if strings.Contains(filepath.Base(dir), projectMatch(name, projects[name])) {
				matches = append(matches, name)
			}
		}
		if len(matches) == 1 {
			name := matches[0]
			return Selection{Name: name, Root: dir, Config: projects[name]}, nil
		}
		if len(matches) > 1 {
			return Selection{}, fmt.Errorf("ambiguous projects at %q: %s; use --project", dir, strings.Join(matches, ", "))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return Selection{}, fmt.Errorf("no configured project matches %q; configured projects: %s; use --project", cleanCWD, projectNames(projects))
}

func matchingAncestor(cwd, match string) (string, bool) {
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if strings.Contains(filepath.Base(dir), match) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
	}
}

func projectMatch(name string, projectConfig config.ProjectConfig) string {
	if projectConfig.Match != "" {
		return projectConfig.Match
	}
	return name
}

func projectNames(projects map[string]config.ProjectConfig) string {
	names := sortedProjectNames(projects)
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}

func sortedProjectNames(projects map[string]config.ProjectConfig) []string {
	names := make([]string, 0, len(projects))
	for name := range projects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
