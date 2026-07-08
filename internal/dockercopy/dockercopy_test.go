package dockercopy

import "testing"

func TestResolveCopyUsesTargetAliasAndSourceRepository(t *testing.T) {
	registries := map[string]string{
		"it": "source-registry.example.com",
		"t":  "mirror-registry.example.com",
	}

	got, err := Resolve(registries, "it/apps/example-service:v1.2.4", "t")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantSource := "source-registry.example.com/apps/example-service:v1.2.4"
	wantDestination := "mirror-registry.example.com/apps/example-service:v1.2.4"
	if got.Source != wantSource || got.Destination != wantDestination {
		t.Fatalf("Resolve() = %#v, want source %q destination %q", got, wantSource, wantDestination)
	}
}

func TestResolveCopyUsesExplicitTargetRepository(t *testing.T) {
	registries := map[string]string{
		"it": "source-registry.example.com",
		"t":  "mirror-registry.example.com",
	}

	got, err := Resolve(registries, "it/apps/example-service:v1.2.4", "t/library/example-service:latest")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantDestination := "mirror-registry.example.com/library/example-service:latest"
	if got.Destination != wantDestination {
		t.Fatalf("destination = %q, want %q", got.Destination, wantDestination)
	}
}

func TestResolveCopyAcceptsLiteralRegistryDomains(t *testing.T) {
	got, err := Resolve(nil, "registry.example.com/apps/example-service:v1.2.4", "mirror.example.com")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantSource := "registry.example.com/apps/example-service:v1.2.4"
	wantDestination := "mirror.example.com/apps/example-service:v1.2.4"
	if got.Source != wantSource || got.Destination != wantDestination {
		t.Fatalf("Resolve() = %#v, want source %q destination %q", got, wantSource, wantDestination)
	}
}

func TestResolveCopyTreatsUnmatchedSourceAliasAsFullImage(t *testing.T) {
	registries := map[string]string{
		"t": "mirror-registry.example.com",
	}

	got, err := Resolve(registries, "python:3.12", "t")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantSource := "docker.io/library/python:3.12"
	wantDestination := "mirror-registry.example.com/python:3.12"
	if got.Source != wantSource || got.Destination != wantDestination {
		t.Fatalf("Resolve() = %#v, want source %q destination %q", got, wantSource, wantDestination)
	}
}

func TestResolveCopyAcceptsFullImageDestinationWithoutAlias(t *testing.T) {
	got, err := Resolve(nil, "python:3.12", "registry.example.com/example-user/python:3.12")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantSource := "docker.io/library/python:3.12"
	wantDestination := "registry.example.com/example-user/python:3.12"
	if got.Source != wantSource || got.Destination != wantDestination {
		t.Fatalf("Resolve() = %#v, want source %q destination %q", got, wantSource, wantDestination)
	}
}

func TestResolveCopyReusesFullSourceRepositoryForRegistryOnlyDestination(t *testing.T) {
	registries := map[string]string{
		"t": "mirror-registry.example.com",
	}

	got, err := Resolve(registries, "registry.example.com/example-user/python:3.12", "t")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantSource := "registry.example.com/example-user/python:3.12"
	wantDestination := "mirror-registry.example.com/example-user/python:3.12"
	if got.Source != wantSource || got.Destination != wantDestination {
		t.Fatalf("Resolve() = %#v, want source %q destination %q", got, wantSource, wantDestination)
	}
}

func TestResolveCopyNormalizesSingleSegmentSourceToDockerHubLibrary(t *testing.T) {
	got, err := Resolve(nil, "python:3.12", "registry.example.com/python:3.12")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	wantSource := "docker.io/library/python:3.12"
	if got.Source != wantSource {
		t.Fatalf("source = %q, want %q", got.Source, wantSource)
	}
}

func TestResolveCopyRejectsUnknownAlias(t *testing.T) {
	_, err := Resolve(map[string]string{"it": "registry.example.com"}, "it/apps/example-service:v1.2.4", "missing")
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	want := `unknown registry alias "missing"; configure docker.registries.missing`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}
