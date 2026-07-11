package logview

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBuildCatalogDiscoversSupportedLogsRecursively(t *testing.T) {
	projectRoot := t.TempDir()
	logRoot := filepath.Join(projectRoot, "runtime", "logs")
	names := []string{
		"api/error.log",
		"api/access.log.2026-07-11",
		"worker.out",
		"worker.err",
		"events.tsv",
		"events.jsonl",
		"events.ndjson",
	}
	for _, name := range names {
		writeTestFile(t, filepath.Join(logRoot, name), []byte("hello\n"))
	}
	for _, name := range []string{"notes.txt", "archive.log.gz", "archive.zip", ".hidden.log", ".cache/debug.log"} {
		writeTestFile(t, filepath.Join(logRoot, name), []byte("skip\n"))
	}
	writeTestFile(t, filepath.Join(logRoot, "binary.log"), []byte{'a', 0, 'b'})
	outside := filepath.Join(projectRoot, "outside.log")
	writeTestFile(t, outside, []byte("outside\n"))
	if err := os.Symlink(outside, filepath.Join(logRoot, "linked.log")); err != nil {
		t.Fatal(err)
	}

	catalog, warnings, err := BuildCatalog(projectRoot, []string{"runtime/logs"})
	if err != nil {
		t.Fatalf("BuildCatalog() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}

	files := catalog.Files()
	got := make([]string, 0, len(files))
	for _, file := range files {
		got = append(got, filepath.ToSlash(file.RelativePath))
		if file.ID == "" || file.Size == 0 || file.Root == "" {
			t.Fatalf("incomplete file metadata: %#v", file)
		}
	}
	sort.Strings(got)
	sort.Strings(names)
	if strings.Join(got, "\n") != strings.Join(names, "\n") {
		t.Fatalf("files = %v, want %v", got, names)
	}
}

func TestBuildCatalogAcceptsAbsoluteRootAndWarnsForInvalidRoots(t *testing.T) {
	projectRoot := t.TempDir()
	valid := filepath.Join(t.TempDir(), "logs")
	writeTestFile(t, filepath.Join(valid, "app.log"), []byte("hello\n"))
	notDirectory := filepath.Join(projectRoot, "single.log")
	writeTestFile(t, notDirectory, []byte("hello\n"))

	catalog, warnings, err := BuildCatalog(projectRoot, []string{"missing", notDirectory, valid})
	if err != nil {
		t.Fatalf("BuildCatalog() error = %v", err)
	}
	if got := len(catalog.Files()); got != 1 {
		t.Fatalf("file count = %d, want 1", got)
	}
	canonicalValid, err := filepath.EvalSymlinks(valid)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(catalog.Roots()); got != 1 || catalog.Roots()[0] != canonicalValid {
		t.Fatalf("roots = %v, want [%s]", catalog.Roots(), canonicalValid)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v, want two", warnings)
	}
}

func TestBuildCatalogFailsWhenEveryRootIsInvalid(t *testing.T) {
	_, warnings, err := BuildCatalog(t.TempDir(), []string{"missing"})
	if err == nil || !strings.Contains(err.Error(), "no usable log directories") {
		t.Fatalf("BuildCatalog() error = %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestCatalogResolveRejectsSymlinkReplacementOutsideRoot(t *testing.T) {
	projectRoot := t.TempDir()
	root := filepath.Join(projectRoot, "logs")
	path := filepath.Join(root, "app.log")
	writeTestFile(t, path, []byte("inside\n"))
	catalog, _, err := BuildCatalog(projectRoot, []string{"logs"})
	if err != nil {
		t.Fatal(err)
	}
	file := catalog.Files()[0]

	outside := filepath.Join(t.TempDir(), "secret.log")
	writeTestFile(t, outside, []byte("secret\n"))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}

	_, err = catalog.Resolve(file.ID)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Resolve() error = %v, want symlink rejection", err)
	}
}

func TestCatalogResolveRejectsUnknownID(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app.log"), []byte("hello\n"))
	catalog, _, err := BuildCatalog(root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	_, err = catalog.Resolve("missing")
	if err == nil || !strings.Contains(err.Error(), "unknown file") {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestValidateOpenedFileRejectsDifferentDescriptor(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "app.log")
	outside := filepath.Join(t.TempDir(), "secret.log")
	writeTestFile(t, inside, []byte("inside\n"))
	writeTestFile(t, outside, []byte("outside\n"))
	catalog, _, err := BuildCatalog(root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := os.Open(outside)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()

	err = validateOpenedFile(opened, catalog.Files()[0])
	if err == nil || !strings.Contains(err.Error(), "changed while opening") {
		t.Fatalf("validateOpenedFile() error = %v", err)
	}
}

func TestCatalogOpenReturnsValidatedFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app.log"), []byte("inside\n"))
	catalog, _, err := BuildCatalog(root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	file, opened, err := catalog.Open(catalog.Files()[0].ID)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer opened.Close()
	contents, err := io.ReadAll(opened)
	if err != nil {
		t.Fatal(err)
	}
	if file.Path == "" || string(contents) != "inside\n" {
		t.Fatalf("file/contents = %#v / %q", file, contents)
	}
}

func writeTestFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatal(err)
	}
}
