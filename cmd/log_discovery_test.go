package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var expectedConventionalLogRoots = []string{
	"runtime/logs",
	"logs",
	"log",
	"var/log",
	"storage/logs",
}

func TestDiscoverLogRootsPrefersAllConventionalDirectories(t *testing.T) {
	root := t.TempDir()
	for _, relative := range expectedConventionalLogRoots {
		mustMkdirAll(t, filepath.Join(root, relative))
	}
	mustMkdirAll(t, filepath.Join(root, "service", "logs"))

	got, warnings := discoverLogRoots(root)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	want := make([]string, 0, len(expectedConventionalLogRoots))
	for _, relative := range expectedConventionalLogRoots {
		want = append(want, filepath.Join(root, relative))
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverLogRoots() = %v, want %v", got, want)
	}
}

func TestDiscoverLogRootsBoundsAndSkipsTraversal(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{
		"app/Logs",
		"a/b/c/logs",
		"a/b/c/d/logs",
		"node_modules/pkg/logs",
		"vendor/pkg/log",
		".cache/logs",
		"dist/logs",
		"build/logs",
		"target/logs",
		".venv/logs",
		"venv/logs",
		"__pycache__/logs",
	} {
		mustMkdirAll(t, filepath.Join(root, relative))
	}

	got, warnings := discoverLogRoots(root)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	want := []string{
		filepath.Join(root, "a", "b", "c", "logs"),
		filepath.Join(root, "app", "Logs"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverLogRoots() = %v, want %v", got, want)
	}
}

func TestDiscoverLogRootsStopsInsideAcceptedRoot(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "service", "logs", "nested", "logs"))
	mustMkdirAll(t, filepath.Join(root, "worker", "LOG"))

	got, warnings := discoverLogRoots(root)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	want := []string{
		filepath.Join(root, "service", "logs"),
		filepath.Join(root, "worker", "LOG"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverLogRoots() = %v, want %v", got, want)
	}
}

func TestDiscoverLogRootsWarnsAboutInvalidConventionalPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "logs"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, warnings := discoverLogRoots(root)
	if len(got) != 0 {
		t.Fatalf("roots = %v, want none", got)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "logs") || !strings.Contains(warnings[0], "not a directory") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestExpandExplicitLogRoots(t *testing.T) {
	cwd := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RELATIVE_LOG_ROOT", "env/logs")
	absolute := filepath.Join(t.TempDir(), "absolute-logs")

	got, err := expandExplicitLogRoots(cwd, []string{
		"./runtime/logs",
		absolute,
		"~/home-logs",
		"$RELATIVE_LOG_ROOT",
		"./runtime/logs",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(cwd, "env", "logs"),
		filepath.Join(cwd, "runtime", "logs"),
		filepath.Join(home, "home-logs"),
		absolute,
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandExplicitLogRoots() = %v, want %v", got, want)
	}
}

func TestExpandExplicitLogRootsRejectsEmptyValue(t *testing.T) {
	_, err := expandExplicitLogRoots(t.TempDir(), []string{"  "})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expandExplicitLogRoots() error = %v, want empty-path error", err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
