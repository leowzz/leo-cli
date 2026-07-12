package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leo/leo-cli/internal/config"
)

func TestResolveFindsProjectFromNestedDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_01-service")
	cwd := filepath.Join(root, "app", "handlers")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"demo_01": {Logs: []string{"runtime/logs"}},
	}

	got, err := Resolve(cwd, "", projects)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Name != "demo_01" || got.Root != root {
		t.Fatalf("Resolve() = %#v, want name demo_01 root %q", got, root)
	}
	if len(got.Config.Logs) != 1 || got.Config.Logs[0] != "runtime/logs" {
		t.Fatalf("config = %#v", got.Config)
	}
}

func TestResolveUsesCustomMatch(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_01-api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"mc": {Match: "demo_01", Logs: []string{"logs"}},
	}

	got, err := Resolve(root, "", projects)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Name != "mc" || got.Root != root {
		t.Fatalf("Resolve() = %#v", got)
	}
}

func TestResolveSelectsNearestAncestor(t *testing.T) {
	outer := filepath.Join(t.TempDir(), "platform")
	inner := filepath.Join(outer, "demo_01-api")
	cwd := filepath.Join(inner, "pkg")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"platform": {Logs: []string{"logs"}},
		"demo_01":  {Logs: []string{"runtime/logs"}},
	}

	got, err := Resolve(cwd, "", projects)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Name != "demo_01" || got.Root != inner {
		t.Fatalf("Resolve() = %#v, want nearest demo_01 at %q", got, inner)
	}
}

func TestResolveRejectsAmbiguousNearestAncestor(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_01-api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"mc":      {Match: "demo_01"},
		"demo_01": {},
	}

	_, err := Resolve(root, "", projects)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("Resolve() error = %v, want ambiguity", err)
	}
}

func TestResolveRequiresMatch(t *testing.T) {
	cwd := t.TempDir()
	projects := map[string]config.ProjectConfig{"demo_01": {}}

	_, err := Resolve(cwd, "", projects)
	if err == nil || !strings.Contains(err.Error(), "demo_01") || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("Resolve() error = %v, want configured projects and --project hint", err)
	}
}

func TestResolveReturnsTypedNoMatch(t *testing.T) {
	_, err := Resolve(t.TempDir(), "", map[string]config.ProjectConfig{"other": {}})
	if !errors.Is(err, ErrNoMatch) {
		t.Fatalf("Resolve() error = %v, want ErrNoMatch", err)
	}
}

func TestResolveDoesNotTypeExplicitOrAmbiguousErrorsAsNoMatch(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"one": {Match: "shared"},
		"two": {Match: "shared"},
	}
	if _, err := Resolve(root, "", projects); err == nil || errors.Is(err, ErrNoMatch) {
		t.Fatalf("ambiguous Resolve() error = %v, want non-ErrNoMatch", err)
	}
	if _, err := Resolve(root, "missing", projects); err == nil || errors.Is(err, ErrNoMatch) {
		t.Fatalf("explicit Resolve() error = %v, want non-ErrNoMatch", err)
	}
}

func TestFindRootUsesNearestGitDirectory(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "service", "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(nested)
	if err != nil || got != root {
		t.Fatalf("FindRoot() = %q, %v, want %q", got, err, root)
	}
}

func TestFindRootUsesNearestGitFile(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "worktree", "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: ../repo/.git/worktrees/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(nested)
	if err != nil || got != root {
		t.Fatalf("FindRoot() = %q, %v, want %q", got, err, root)
	}
}

func TestFindRootFallsBackToCurrentDirectory(t *testing.T) {
	cwd := t.TempDir()
	got, err := FindRoot(cwd)
	if err != nil || got != cwd {
		t.Fatalf("FindRoot() = %q, %v, want %q", got, err, cwd)
	}
}

func TestResolveExplicitProjectStillRequiresMatchingAncestor(t *testing.T) {
	cwd := t.TempDir()
	projects := map[string]config.ProjectConfig{
		"mc": {Match: "demo_01"},
	}

	_, err := Resolve(cwd, "mc", projects)
	if err == nil || !strings.Contains(err.Error(), "not inside") {
		t.Fatalf("Resolve() error = %v, want root safety error", err)
	}
}

func TestResolveExplicitProjectBreaksAmbiguity(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_01-api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"mc":      {Match: "demo_01", Logs: []string{"short/logs"}},
		"demo_01": {Logs: []string{"long/logs"}},
	}

	got, err := Resolve(root, "mc", projects)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Name != "mc" || got.Config.Logs[0] != "short/logs" {
		t.Fatalf("Resolve() = %#v", got)
	}
}

func TestResolveRejectsUnknownExplicitProject(t *testing.T) {
	_, err := Resolve(t.TempDir(), "missing", map[string]config.ProjectConfig{"demo_01": {}})
	if err == nil || !strings.Contains(err.Error(), "unknown project") {
		t.Fatalf("Resolve() error = %v, want unknown project", err)
	}
}
