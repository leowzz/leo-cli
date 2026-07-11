package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leo/leo-cli/internal/config"
)

func TestResolveFindsProjectFromNestedDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mindcraft-service")
	cwd := filepath.Join(root, "app", "handlers")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"mindcraft": {Logs: []string{"runtime/logs"}},
	}

	got, err := Resolve(cwd, "", projects)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Name != "mindcraft" || got.Root != root {
		t.Fatalf("Resolve() = %#v, want name mindcraft root %q", got, root)
	}
	if len(got.Config.Logs) != 1 || got.Config.Logs[0] != "runtime/logs" {
		t.Fatalf("config = %#v", got.Config)
	}
}

func TestResolveUsesCustomMatch(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mindcraft-api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"mc": {Match: "mindcraft", Logs: []string{"logs"}},
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
	inner := filepath.Join(outer, "mindcraft-api")
	cwd := filepath.Join(inner, "pkg")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"platform":  {Logs: []string{"logs"}},
		"mindcraft": {Logs: []string{"runtime/logs"}},
	}

	got, err := Resolve(cwd, "", projects)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Name != "mindcraft" || got.Root != inner {
		t.Fatalf("Resolve() = %#v, want nearest mindcraft at %q", got, inner)
	}
}

func TestResolveRejectsAmbiguousNearestAncestor(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mindcraft-api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"mc":        {Match: "mindcraft"},
		"mindcraft": {},
	}

	_, err := Resolve(root, "", projects)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("Resolve() error = %v, want ambiguity", err)
	}
}

func TestResolveRequiresMatch(t *testing.T) {
	cwd := t.TempDir()
	projects := map[string]config.ProjectConfig{"mindcraft": {}}

	_, err := Resolve(cwd, "", projects)
	if err == nil || !strings.Contains(err.Error(), "mindcraft") || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("Resolve() error = %v, want configured projects and --project hint", err)
	}
}

func TestResolveExplicitProjectStillRequiresMatchingAncestor(t *testing.T) {
	cwd := t.TempDir()
	projects := map[string]config.ProjectConfig{
		"mc": {Match: "mindcraft"},
	}

	_, err := Resolve(cwd, "mc", projects)
	if err == nil || !strings.Contains(err.Error(), "not inside") {
		t.Fatalf("Resolve() error = %v, want root safety error", err)
	}
}

func TestResolveExplicitProjectBreaksAmbiguity(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mindcraft-api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	projects := map[string]config.ProjectConfig{
		"mc":        {Match: "mindcraft", Logs: []string{"short/logs"}},
		"mindcraft": {Logs: []string{"long/logs"}},
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
	_, err := Resolve(t.TempDir(), "missing", map[string]config.ProjectConfig{"mindcraft": {}})
	if err == nil || !strings.Contains(err.Error(), "unknown project") {
		t.Fatalf("Resolve() error = %v, want unknown project", err)
	}
}
