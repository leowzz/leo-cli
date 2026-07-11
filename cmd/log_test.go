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
	if hostFlag == nil || hostFlag.DefValue != "127.0.0.1" {
		t.Fatalf("host default = %#v", hostFlag)
	}
	if portFlag == nil || portFlag.DefValue != "0" {
		t.Fatalf("port default = %#v", portFlag)
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

	runtime, warnings, err := prepareLogRuntime(cfg, cwd, "")
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
}

func TestPrepareLogRuntimeRejectsMissingConfiguration(t *testing.T) {
	_, _, err := prepareLogRuntime(config.Config{}, t.TempDir(), "")
	if err == nil || !strings.Contains(err.Error(), "configured projects") {
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
	}}, root, "demo_01")
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
