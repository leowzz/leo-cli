package cmd

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/leo/leo-cli/internal/config"
)

func wantSkopeoCopyArgs(source, destination, platform string) []string {
	args, err := skopeoCopyArgs(source, destination, platform)
	if err != nil {
		panic(err)
	}
	return args
}

func TestParseCopyPlatform(t *testing.T) {
	got, err := parseCopyPlatform("linux/amd64")
	if err != nil {
		t.Fatalf("parseCopyPlatform() error = %v", err)
	}
	want := copyPlatform{os: "linux", arch: "amd64"}
	if got != want {
		t.Fatalf("parseCopyPlatform() = %#v, want %#v", got, want)
	}

	got, err = parseCopyPlatform("linux/arm64/v8")
	if err != nil {
		t.Fatalf("parseCopyPlatform() error = %v", err)
	}
	want = copyPlatform{os: "linux", arch: "arm64", variant: "v8"}
	if got != want {
		t.Fatalf("parseCopyPlatform() = %#v, want %#v", got, want)
	}

	if _, err := parseCopyPlatform("linux"); err == nil {
		t.Fatal("parseCopyPlatform() error = nil, want error")
	}
}

func TestSkopeoCopyArgsIncludesPlatformOverride(t *testing.T) {
	got, err := skopeoCopyArgs("docker.io/library/python:3.12-slim", "registry.example.com/python:3.12-slim", "linux/amd64")
	if err != nil {
		t.Fatalf("skopeoCopyArgs() error = %v", err)
	}
	want := []string{
		"copy",
		"--override-os", "linux",
		"--override-arch", "amd64",
		"docker://docker.io/library/python:3.12-slim",
		"docker://registry.example.com/python:3.12-slim",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("skopeoCopyArgs() = %#v, want %#v", got, want)
	}
}

func TestRunDockerCopyInvokesSkopeoWithResolvedReferences(t *testing.T) {
	cfg := config.Config{
		Docker: config.DockerConfig{
			Registries: map[string]string{
				"it": "source-registry.example.com",
				"t":  "mirror-registry.example.com",
			},
		},
	}

	var gotName string
	var gotArgs []string
	runner := func(_ context.Context, name string, args []string, _, _ io.Writer) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	err := runDockerCopy(context.Background(), cfg, "it/apps/example-service:v1.2.4", "t/library/example-service:latest", defaultDockerCopyPlatform, false, io.Discard, io.Discard, runner)
	if err != nil {
		t.Fatalf("runDockerCopy() error = %v", err)
	}

	if gotName != "skopeo" {
		t.Fatalf("command = %q, want %q", gotName, "skopeo")
	}

	wantArgs := wantSkopeoCopyArgs(
		"source-registry.example.com/apps/example-service:v1.2.4",
		"mirror-registry.example.com/library/example-service:latest",
		defaultDockerCopyPlatform,
	)
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestRunDockerCopyReusesSourceRepositoryForRegistryOnlyDestination(t *testing.T) {
	cfg := config.Config{
		Docker: config.DockerConfig{
			Registries: map[string]string{
				"it": "source-registry.example.com",
				"t":  "mirror-registry.example.com",
			},
		},
	}

	var gotArgs []string
	runner := func(_ context.Context, _ string, args []string, _, _ io.Writer) error {
		gotArgs = append([]string(nil), args...)
		return nil
	}

	err := runDockerCopy(context.Background(), cfg, "it/apps/example-service:v1.2.4", "t", defaultDockerCopyPlatform, false, io.Discard, io.Discard, runner)
	if err != nil {
		t.Fatalf("runDockerCopy() error = %v", err)
	}

	wantArgs := wantSkopeoCopyArgs(
		"source-registry.example.com/apps/example-service:v1.2.4",
		"mirror-registry.example.com/apps/example-service:v1.2.4",
		defaultDockerCopyPlatform,
	)
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestRunDockerCopyDryRunPrintsRenderedCommandWithoutRunning(t *testing.T) {
	cfg := config.Config{
		Docker: config.DockerConfig{
			Registries: map[string]string{
				"it": "source-registry.example.com",
				"t":  "mirror-registry.example.com",
			},
		},
	}

	called := false
	runner := func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
		called = true
		return nil
	}

	var stdout bytes.Buffer
	err := runDockerCopy(context.Background(), cfg, "it/apps/example-service:v1.2.4", "t", defaultDockerCopyPlatform, true, &stdout, io.Discard, runner)
	if err != nil {
		t.Fatalf("runDockerCopy() error = %v", err)
	}
	if called {
		t.Fatal("runner was called in dry run")
	}

	want := renderCommand("skopeo", wantSkopeoCopyArgs(
		"source-registry.example.com/apps/example-service:v1.2.4",
		"mirror-registry.example.com/apps/example-service:v1.2.4",
		defaultDockerCopyPlatform,
	)) + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunDockerCopyDryRunAcceptsFullSourceImage(t *testing.T) {
	cfg := config.Config{
		Docker: config.DockerConfig{
			Registries: map[string]string{
				"t": "mirror-registry.example.com",
			},
		},
	}

	called := false
	runner := func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
		called = true
		return nil
	}

	var stdout bytes.Buffer
	err := runDockerCopy(context.Background(), cfg, "python:3.12", "t", defaultDockerCopyPlatform, true, &stdout, io.Discard, runner)
	if err != nil {
		t.Fatalf("runDockerCopy() error = %v", err)
	}
	if called {
		t.Fatal("runner was called in dry run")
	}

	want := renderCommand("skopeo", wantSkopeoCopyArgs("docker.io/library/python:3.12", "mirror-registry.example.com/python:3.12", defaultDockerCopyPlatform)) + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunDockerCopyDryRunAcceptsFullSourceAndDestinationImages(t *testing.T) {
	cfg := config.Config{}

	called := false
	runner := func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
		called = true
		return nil
	}

	var stdout bytes.Buffer
	err := runDockerCopy(context.Background(), cfg, "python:3.12", "registry.example.com/example-user/python:3.12", defaultDockerCopyPlatform, true, &stdout, io.Discard, runner)
	if err != nil {
		t.Fatalf("runDockerCopy() error = %v", err)
	}
	if called {
		t.Fatal("runner was called in dry run")
	}

	want := renderCommand("skopeo", wantSkopeoCopyArgs("docker.io/library/python:3.12", "registry.example.com/example-user/python:3.12", defaultDockerCopyPlatform)) + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunDockerListPrintsConfiguredAliasesSorted(t *testing.T) {
	cfg := config.Config{
		Docker: config.DockerConfig{
			Registries: map[string]string{
				"t":  "mirror-registry.example.com",
				"it": "source-registry.example.com",
			},
		},
	}

	var stdout bytes.Buffer
	if err := runDockerList(cfg, &stdout); err != nil {
		t.Fatalf("runDockerList() error = %v", err)
	}

	want := "it\tsource-registry.example.com\n" +
		"t\tmirror-registry.example.com\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunDockerListPrintsEmptyMessage(t *testing.T) {
	var stdout bytes.Buffer
	if err := runDockerList(config.Config{}, &stdout); err != nil {
		t.Fatalf("runDockerList() error = %v", err)
	}

	want := "No Docker registries configured.\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}
