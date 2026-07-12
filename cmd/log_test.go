package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leo/leo-cli/internal/config"
)

func TestLogCommandDefaults(t *testing.T) {
	if got, want := logCmd.Use, "log"; got != want {
		t.Fatalf("logCmd.Use = %q, want %q", got, want)
	}
	hostFlag := logCmd.Flags().Lookup("host")
	portFlag := logCmd.Flags().Lookup("port")
	logsFlag := logCmd.Flags().Lookup("logs")
	if hostFlag == nil || hostFlag.DefValue != "127.0.0.1" {
		t.Fatalf("host default = %#v", hostFlag)
	}
	if portFlag == nil || portFlag.DefValue != "0" {
		t.Fatalf("port default = %#v", portFlag)
	}
	if logsFlag == nil || logsFlag.DefValue != "[]" {
		t.Fatalf("logs default = %#v", logsFlag)
	}
}

func TestPrepareLogRuntimeResolvesProjectAndRelativeRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_01-api")
	cwd := filepath.Join(root, "app", "handlers")
	logs := filepath.Join(root, "runtime", "logs")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "app.log"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Projects: map[string]config.ProjectConfig{
		"demo_01": {Logs: []string{"runtime/logs", "missing"}},
	}}

	runtime, warnings, err := prepareLogRuntime(cfg, cwd, "", nil)
	if err != nil {
		t.Fatalf("prepareLogRuntime() error = %v", err)
	}
	if runtime.project.Name != "demo_01" || runtime.project.Root != root {
		t.Fatalf("project = %#v", runtime.project)
	}
	if len(runtime.catalog.Files()) != 1 || len(runtime.catalog.Roots()) != 1 {
		t.Fatalf("catalog files/roots = %#v / %#v", runtime.catalog.Files(), runtime.catalog.Roots())
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "missing") {
		t.Fatalf("warnings = %v", warnings)
	}
	if runtime.automatic {
		t.Fatal("configured runtime marked automatic")
	}
}

func TestPrepareLogRuntimeDiscoversLogsWithoutConfiguration(t *testing.T) {
	root := filepath.Join(t.TempDir(), "first-run-service")
	cwd := filepath.Join(root, "app", "handlers")
	logs := filepath.Join(root, "runtime", "logs")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "error.log"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runtime, warnings, err := prepareLogRuntime(config.Config{}, cwd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 || !runtime.automatic || runtime.project.Root != root || runtime.project.Name != filepath.Base(root) {
		t.Fatalf("runtime/warnings = %#v / %v", runtime, warnings)
	}
	if got := runtime.catalog.Roots(); len(got) != 1 || got[0] != canonicalTestPath(t, logs) {
		t.Fatalf("catalog roots = %v, want %q", got, logs)
	}
	if len(runtime.catalog.Files()) != 1 {
		t.Fatalf("catalog files = %#v", runtime.catalog.Files())
	}
}

func TestPrepareLogRuntimeFallsBackWhenConfiguredProjectsDoNotMatch(t *testing.T) {
	root := filepath.Join(t.TempDir(), "new-service")
	logs := filepath.Join(root, "logs")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "console.log"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Projects: map[string]config.ProjectConfig{"other": {Logs: []string{"logs"}}}}

	runtime, _, err := prepareLogRuntime(cfg, root, "", nil)
	if err != nil || !runtime.automatic {
		t.Fatalf("prepareLogRuntime() = %#v, %v", runtime, err)
	}
}

func TestPrepareLogRuntimePrefersMatchingConfiguration(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_01-service")
	configuredLogs := filepath.Join(root, "configured")
	autoLogs := filepath.Join(root, "runtime", "logs")
	for _, dir := range []string{configuredLogs, autoLogs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "app.log"), []byte("hello\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Config{Projects: map[string]config.ProjectConfig{"demo_01": {Logs: []string{"configured"}}}}

	runtime, _, err := prepareLogRuntime(cfg, root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.automatic || runtime.catalog.Roots()[0] != canonicalTestPath(t, configuredLogs) {
		t.Fatalf("runtime = %#v", runtime)
	}
}

func TestPrepareLogRuntimeUsesExplicitLogRoots(t *testing.T) {
	root := t.TempDir()
	custom := filepath.Join(root, "custom", "logs")
	autoLogs := filepath.Join(root, "runtime", "logs")
	for _, dir := range []string{custom, autoLogs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "app.log"), []byte("hello\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	runtime, _, err := prepareLogRuntime(config.Config{}, root, "", []string{"./custom/logs"})
	if err != nil {
		t.Fatal(err)
	}
	if !runtime.automatic || len(runtime.catalog.Roots()) != 1 || runtime.catalog.Roots()[0] != canonicalTestPath(t, custom) {
		t.Fatalf("runtime = %#v", runtime)
	}
}

func TestPrepareLogRuntimeRejectsProjectWithExplicitLogRoots(t *testing.T) {
	_, _, err := prepareLogRuntime(config.Config{}, t.TempDir(), "configured", []string{"logs"})
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("prepareLogRuntime() error = %v", err)
	}
}

func TestPrepareLogRuntimeKeepsExplicitProjectStrict(t *testing.T) {
	_, _, err := prepareLogRuntime(config.Config{}, t.TempDir(), "missing", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown project") {
		t.Fatalf("prepareLogRuntime() error = %v", err)
	}
}

func TestPrepareLogRuntimeExplainsMissingAutomaticLogs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "empty-service")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, err := prepareLogRuntime(config.Config{}, root, "", nil)
	if err == nil {
		t.Fatal("prepareLogRuntime() error = nil")
	}
	for _, want := range []string{root, "depth 4", "leo log --logs ./path/to/logs", "proj:", filepath.Base(root)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q:\n%s", want, err)
		}
	}
}

func TestPrepareLogRuntimeExplainsUnusableExplicitRoots(t *testing.T) {
	root := t.TempDir()
	_, _, err := prepareLogRuntime(config.Config{}, root, "", []string{"./missing"})
	if err == nil || !strings.Contains(err.Error(), "--logs") || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("prepareLogRuntime() error = %v", err)
	}
}

func TestPrintLogStartup(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_01")
	logs := filepath.Join(root, "logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "app.log"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtime, _, err := prepareLogRuntime(config.Config{Projects: map[string]config.ProjectConfig{
		"demo_01": {Logs: []string{"logs"}},
	}}, root, "demo_01", nil)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	printLogStartup(&output, runtime, []string{"skip warning"}, "http://127.0.0.1:9031/bootstrap?token=secret")
	got := output.String()
	for _, want := range []string{"Project: demo_01", "Root: " + root, "Logs:", logs, "Warning: skip warning", "Open: http://127.0.0.1:9031/bootstrap?token=secret"} {
		if !strings.Contains(got, want) {
			t.Errorf("startup output missing %q:\n%s", want, got)
		}
	}
}

func TestPrintLogStartupMarksAutomaticRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auto-service")
	logs := filepath.Join(root, "logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "console.log"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtime, _, err := prepareLogRuntime(config.Config{}, root, "", []string{"logs"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	printLogStartup(&output, runtime, nil, "http://127.0.0.1/bootstrap?token=secret")
	if !strings.Contains(output.String(), "Project: auto-service (auto)") {
		t.Fatalf("startup output = %q", output.String())
	}
}

func TestAdvertisedLogHostUsesHostnameForWildcardBind(t *testing.T) {
	for _, bindHost := range []string{"0.0.0.0", "::"} {
		got, err := advertisedLogHost(bindHost, func() (string, error) { return "devbox.internal", nil })
		if err != nil {
			t.Fatalf("advertisedLogHost(%q) error = %v", bindHost, err)
		}
		if got != "devbox.internal" {
			t.Fatalf("advertisedLogHost(%q) = %q", bindHost, got)
		}
	}
}

func TestAdvertisedLogHostKeepsSpecificBind(t *testing.T) {
	got, err := advertisedLogHost("10.0.0.8", func() (string, error) {
		return "", errors.New("must not be called")
	})
	if err != nil || got != "10.0.0.8" {
		t.Fatalf("advertisedLogHost() = %q, %v", got, err)
	}
}

func canonicalTestPath(t *testing.T, path string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(canonical)
}
