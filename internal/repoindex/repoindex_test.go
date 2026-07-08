package repoindex

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/leo/leo-cli/internal/store"
)

func TestScanRootsFindsNormalRepoWithHeadLogTimestamp(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "alpha")
	logPath := filepath.Join(repoPath, ".git", "logs", "HEAD")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	log := "0000000 1111111 Tester <tester@example.com> 1710000000 +0800\tcommit: first\n" +
		"1111111 2222222 Tester <tester@example.com> 1710000600 +0800\tcommit: second\n"
	if err := os.WriteFile(logPath, []byte(log), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := ScanRoots([]string{root}, time.Unix(2000000000, 0))

	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", result.Warnings)
	}
	if len(result.Repos) != 1 {
		t.Fatalf("repos len = %d, want 1", len(result.Repos))
	}
	repo := result.Repos[0]
	if repo.Path != repoPath {
		t.Fatalf("repo path = %q, want %q", repo.Path, repoPath)
	}
	if repo.Name != "alpha" {
		t.Fatalf("repo name = %q, want alpha", repo.Name)
	}
	if repo.LastGitActivityAt.Unix() != 1710000600 {
		t.Fatalf("activity = %d, want 1710000600", repo.LastGitActivityAt.Unix())
	}
	if repo.CurrentBranch != "" {
		t.Fatalf("branch = %q, want empty", repo.CurrentBranch)
	}
	if !repo.LastCommitAt.IsZero() {
		t.Fatalf("last commit = %v, want zero", repo.LastCommitAt)
	}
	if repo.LastIndexedAt.Unix() != 2000000000 {
		t.Fatalf("indexed = %d, want 2000000000", repo.LastIndexedAt.Unix())
	}
}

func TestScanRootsReadsCurrentBranchAndLastCommitTime(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "alpha")
	gitDir := filepath.Join(repoPath, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(HEAD) error = %v", err)
	}
	commitID := writeCommitObject(t, gitDir, 1740000000)
	refPath := filepath.Join(gitDir, "refs", "heads", "main")
	if err := os.MkdirAll(filepath.Dir(refPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(ref) error = %v", err)
	}
	if err := os.WriteFile(refPath, []byte(commitID+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(ref) error = %v", err)
	}

	result := ScanRoots([]string{root}, time.Unix(2000000000, 0))

	if len(result.Repos) != 1 {
		t.Fatalf("repos len = %d, want 1", len(result.Repos))
	}
	repo := result.Repos[0]
	if repo.CurrentBranch != "main" {
		t.Fatalf("branch = %q, want main", repo.CurrentBranch)
	}
	if repo.LastCommitAt.IsZero() || repo.LastCommitAt.Unix() != 1740000000 {
		t.Fatalf("last commit = %v, want unix 1740000000", repo.LastCommitAt)
	}
}

func TestRefreshReposUpdatesKnownRepoMetadata(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "alpha")
	gitDir := filepath.Join(repoPath, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(HEAD) error = %v", err)
	}
	commitID := writeCommitObject(t, gitDir, 1740000300)
	refPath := filepath.Join(gitDir, "refs", "heads", "feature")
	if err := os.MkdirAll(filepath.Dir(refPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(ref) error = %v", err)
	}
	if err := os.WriteFile(refPath, []byte(commitID+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(ref) error = %v", err)
	}

	result := RefreshRepos([]store.Repo{{
		Path:              repoPath,
		Name:              "alpha",
		CurrentBranch:     "old",
		LastCommitAt:      time.Unix(1, 0),
		LastGitActivityAt: time.Unix(2, 0),
		LastIndexedAt:     time.Unix(3, 0),
	}}, time.Unix(2000000000, 0))

	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", result.Warnings)
	}
	if len(result.Repos) != 1 {
		t.Fatalf("repos len = %d, want 1", len(result.Repos))
	}
	repo := result.Repos[0]
	if repo.Name != "alpha" {
		t.Fatalf("name = %q, want alpha", repo.Name)
	}
	if repo.CurrentBranch != "feature" {
		t.Fatalf("branch = %q, want feature", repo.CurrentBranch)
	}
	if repo.LastCommitAt.Unix() != 1740000300 {
		t.Fatalf("commit = %d, want 1740000300", repo.LastCommitAt.Unix())
	}
	if repo.LastIndexedAt.Unix() != 2000000000 {
		t.Fatalf("indexed = %d, want 2000000000", repo.LastIndexedAt.Unix())
	}
}

func TestScanRootsFindsNestedRepos(t *testing.T) {
	root := t.TempDir()
	parentGit := filepath.Join(root, "parent", ".git")
	childGit := filepath.Join(root, "parent", "nested", ".git")
	if err := os.MkdirAll(parentGit, 0o755); err != nil {
		t.Fatalf("MkdirAll(parent) error = %v", err)
	}
	if err := os.MkdirAll(childGit, 0o755); err != nil {
		t.Fatalf("MkdirAll(child) error = %v", err)
	}

	result := ScanRoots([]string{root}, time.Unix(2000000000, 0))

	if len(result.Repos) != 2 {
		t.Fatalf("repos len = %d, want 2", len(result.Repos))
	}
	got := map[string]bool{}
	for _, repo := range result.Repos {
		got[repo.Path] = true
	}
	if !got[filepath.Join(root, "parent")] {
		t.Fatalf("missing parent repo in %#v", got)
	}
	if !got[filepath.Join(root, "parent", "nested")] {
		t.Fatalf("missing nested repo in %#v", got)
	}
}

func TestScanRootsFindsWorktreeGitFile(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "scan-root")
	repoPath := filepath.Join(root, "worktree")
	gitDir := filepath.Join(base, "real-gitdir", "worktrees", "worktree")
	logPath := filepath.Join(gitDir, "logs", "HEAD")

	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(repo) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(log) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, ".git"), []byte("gitdir: "+gitDir+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.git) error = %v", err)
	}
	log := "0000000 1111111 Tester <tester@example.com> 1720000000 +0800\tcheckout: moving\n"
	if err := os.WriteFile(logPath, []byte(log), 0o644); err != nil {
		t.Fatalf("WriteFile(log) error = %v", err)
	}

	result := ScanRoots([]string{root}, time.Unix(2000000000, 0))

	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", result.Warnings)
	}
	if len(result.Repos) != 1 {
		t.Fatalf("repos len = %d, want 1", len(result.Repos))
	}
	if result.Repos[0].Path != repoPath {
		t.Fatalf("repo path = %q, want %q", result.Repos[0].Path, repoPath)
	}
	if result.Repos[0].LastGitActivityAt.Unix() != 1720000000 {
		t.Fatalf("activity = %d, want 1720000000", result.Repos[0].LastGitActivityAt.Unix())
	}
}

func TestScanRootsIgnoresEmptyGitFile(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "uv-cache", "sdists-v9")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, ".git"), nil, 0o644); err != nil {
		t.Fatalf("WriteFile(.git) error = %v", err)
	}

	result := ScanRoots([]string{root}, time.Unix(2000000000, 0))

	if len(result.Repos) != 0 {
		t.Fatalf("repos = %#v, want none", result.Repos)
	}
}

func TestScanRootsReportsMissingRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	result := ScanRoots([]string{missing}, time.Now())

	if len(result.Repos) != 0 {
		t.Fatalf("repos len = %d, want 0", len(result.Repos))
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(result.Warnings))
	}
}

func TestScanRootsFallsBackToRepoMTime(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "beta")
	gitPath := filepath.Join(repoPath, ".git")
	if err := os.MkdirAll(gitPath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	want := time.Unix(1730000000, 0)
	if err := os.Chtimes(gitPath, want, want); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	result := ScanRoots([]string{root}, time.Unix(2000000000, 0))

	if len(result.Repos) != 1 {
		t.Fatalf("repos len = %d, want 1", len(result.Repos))
	}
	if result.Repos[0].LastGitActivityAt.Unix() != want.Unix() {
		t.Fatalf("activity = %d, want %d", result.Repos[0].LastGitActivityAt.Unix(), want.Unix())
	}
}

func writeCommitObject(t *testing.T, gitDir string, committerUnix int64) string {
	t.Helper()

	body := []byte("tree 0000000000000000000000000000000000000000\n" +
		"author Tester <tester@example.com> 1730000000 +0800\n" +
		"committer Tester <tester@example.com> " + strconv.FormatInt(committerUnix, 10) + " +0800\n\n" +
		"test commit\n")
	raw := append([]byte("commit "+strconv.Itoa(len(body))+"\x00"), body...)
	sum := sha1.Sum(raw)
	id := hex.EncodeToString(sum[:])

	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(raw); err != nil {
		t.Fatalf("zlib write error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("zlib close error = %v", err)
	}

	objectPath := filepath.Join(gitDir, "objects", id[:2], id[2:])
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(object) error = %v", err)
	}
	if err := os.WriteFile(objectPath, compressed.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile(object) error = %v", err)
	}
	return id
}
