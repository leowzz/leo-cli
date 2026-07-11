package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnsureCreatesDefaultConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := Ensure(path); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []string{"~/work"}
	if !reflect.DeepEqual(cfg.Repo.Roots, want) {
		t.Fatalf("roots = %#v, want %#v", cfg.Repo.Roots, want)
	}
	wantZones := []string{"+9", "+0"}
	if !reflect.DeepEqual(cfg.Time.Zones, wantZones) {
		t.Fatalf("time zones = %#v, want %#v", cfg.Time.Zones, wantZones)
	}
}

func TestEnsureDoesNotOverwriteExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := []byte("repo:\n  roots:\n    - /tmp/projects\n")
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := Ensure(path); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(contents) {
		t.Fatalf("config was overwritten:\n%s", got)
	}
}

func TestLoadYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("repo:\n  roots:\n    - ~/work\n    - $PROJECTS\ndocker:\n  registries:\n    it: source-registry.example.com\n    t: mirror-registry.example.com\ntime:\n  zones:\n    - +9\n    - +0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []string{"~/work", "$PROJECTS"}
	if !reflect.DeepEqual(cfg.Repo.Roots, want) {
		t.Fatalf("roots = %#v, want %#v", cfg.Repo.Roots, want)
	}

	wantRegistries := map[string]string{
		"it": "source-registry.example.com",
		"t":  "mirror-registry.example.com",
	}
	if !reflect.DeepEqual(cfg.Docker.Registries, wantRegistries) {
		t.Fatalf("docker registries = %#v, want %#v", cfg.Docker.Registries, wantRegistries)
	}

	wantZones := []string{"+9", "+0"}
	if !reflect.DeepEqual(cfg.Time.Zones, wantZones) {
		t.Fatalf("time zones = %#v, want %#v", cfg.Time.Zones, wantZones)
	}
}

func TestLoadProjects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := "proj:\n  mindcraft:\n    logs:\n      - runtime/logs\n      - /docker-runtime\n  mc:\n    match: mindcraft\n    logs:\n      - var/log\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := map[string]ProjectConfig{
		"mindcraft": {Logs: []string{"runtime/logs", "/docker-runtime"}},
		"mc":        {Match: "mindcraft", Logs: []string{"var/log"}},
	}
	if !reflect.DeepEqual(cfg.Projects, want) {
		t.Fatalf("projects = %#v, want %#v", cfg.Projects, want)
	}
}

func TestLoadWithoutProjectsIsBackwardCompatible(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("repo:\n  roots:\n    - ~/work\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Projects != nil {
		t.Fatalf("projects = %#v, want nil", cfg.Projects)
	}
}

func TestExpandPathExpandsHomeAndEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROJECTS", "src")

	got, err := ExpandPath("~/work/$PROJECTS")
	if err != nil {
		t.Fatalf("ExpandPath() error = %v", err)
	}

	want := filepath.Join(home, "work", "src")
	if got != want {
		t.Fatalf("ExpandPath() = %q, want %q", got, want)
	}
}

func TestExpandedRepoRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROJECTS", "src")

	got, err := ExpandedRepoRoots(Config{
		Repo: RepoConfig{Roots: []string{"~/work", "$HOME/$PROJECTS"}},
	})
	if err != nil {
		t.Fatalf("ExpandedRepoRoots() error = %v", err)
	}

	want := []string{
		filepath.Join(home, "work"),
		filepath.Join(home, "src"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedRepoRoots() = %#v, want %#v", got, want)
	}
}
