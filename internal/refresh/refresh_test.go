package refresh

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leo/leo-cli/internal/store"
)

func TestEnsureInitialIndexRunsFullScanWhenEmpty(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "leo-cli.sqlite3"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	var called bool
	result, err := EnsureInitialIndex(ctx, db, []string{"/tmp/root"}, time.Unix(1000, 0), func(roots []string, now time.Time) ScanResult {
		called = true
		return ScanResult{Repos: []store.Repo{{
			Path:              "/tmp/root/repo",
			Name:              "repo",
			CurrentBranch:     "main",
			LastCommitAt:      time.Unix(900, 0),
			LastGitActivityAt: time.Unix(950, 0),
			LastIndexedAt:     now,
		}}}
	})
	if err != nil {
		t.Fatalf("EnsureInitialIndex() error = %v", err)
	}
	if !called {
		t.Fatalf("scan was not called for empty index")
	}
	if !result.Ran {
		t.Fatalf("Ran = false, want true")
	}

	repos, err := db.ListRepos(ctx)
	if err != nil {
		t.Fatalf("ListRepos() error = %v", err)
	}
	if len(repos) != 1 || repos[0].Path != "/tmp/root/repo" {
		t.Fatalf("repos = %#v, want indexed repo", repos)
	}
}

func TestMaybeRefreshSkipsFreshIndex(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "leo-cli.sqlite3"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()
	if err := db.UpsertRepos(ctx, []store.Repo{{
		Path:              "/tmp/repo",
		Name:              "repo",
		LastGitActivityAt: time.Unix(900, 0),
		LastIndexedAt:     time.Unix(995, 0),
	}}); err != nil {
		t.Fatalf("UpsertRepos() error = %v", err)
	}

	var called bool
	result, err := MaybeRefresh(ctx, db, Options{
		Roots:        []string{"/tmp/root"},
		Now:          time.Unix(1000, 0),
		TTL:          time.Minute,
		LockPath:     filepath.Join(t.TempDir(), "repo-refresh.lock"),
		MetadataOnly: false,
		Scan: func(roots []string, now time.Time) ScanResult {
			called = true
			return ScanResult{}
		},
	})
	if err != nil {
		t.Fatalf("MaybeRefresh() error = %v", err)
	}
	if called {
		t.Fatalf("scan was called for fresh index")
	}
	if result.Ran || result.SkippedReason != "fresh" {
		t.Fatalf("result = %#v, want fresh skip", result)
	}
}

func TestMaybeRefreshRunsWhenStaleAndUsesLock(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "leo-cli.sqlite3"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()
	if err := db.UpsertRepos(ctx, []store.Repo{{
		Path:              "/tmp/repo",
		Name:              "repo",
		LastGitActivityAt: time.Unix(100, 0),
		LastIndexedAt:     time.Unix(100, 0),
	}}); err != nil {
		t.Fatalf("UpsertRepos() error = %v", err)
	}

	lockPath := filepath.Join(t.TempDir(), "repo-refresh.lock")
	var called bool
	result, err := MaybeRefresh(ctx, db, Options{
		Roots:    []string{"/tmp/root"},
		Now:      time.Unix(1000, 0),
		TTL:      time.Minute,
		LockPath: lockPath,
		Scan: func(roots []string, now time.Time) ScanResult {
			called = true
			if _, err := os.Stat(lockPath); err != nil {
				t.Fatalf("lock did not exist during scan: %v", err)
			}
			return ScanResult{Repos: []store.Repo{{
				Path:              "/tmp/root/new",
				Name:              "new",
				LastGitActivityAt: time.Unix(999, 0),
				LastIndexedAt:     now,
			}}}
		},
	})
	if err != nil {
		t.Fatalf("MaybeRefresh() error = %v", err)
	}
	if !called || !result.Ran {
		t.Fatalf("called = %v result = %#v, want scan run", called, result)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock still exists after refresh, stat err = %v", err)
	}
}

func TestMaybeRefreshSkipsWhenLocked(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "leo-cli.sqlite3"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()
	if err := db.UpsertRepos(ctx, []store.Repo{{
		Path:              "/tmp/repo",
		Name:              "repo",
		LastGitActivityAt: time.Unix(100, 0),
		LastIndexedAt:     time.Unix(100, 0),
	}}); err != nil {
		t.Fatalf("UpsertRepos() error = %v", err)
	}

	lockPath := filepath.Join(t.TempDir(), "repo-refresh.lock")
	if err := os.WriteFile(lockPath, []byte("locked"), 0o644); err != nil {
		t.Fatalf("WriteFile(lock) error = %v", err)
	}
	var called bool
	result, err := MaybeRefresh(ctx, db, Options{
		Roots:    []string{"/tmp/root"},
		Now:      time.Unix(1000, 0),
		TTL:      time.Minute,
		LockPath: lockPath,
		Scan: func(roots []string, now time.Time) ScanResult {
			called = true
			return ScanResult{}
		},
	})
	if err != nil {
		t.Fatalf("MaybeRefresh() error = %v", err)
	}
	if called {
		t.Fatalf("scan was called while locked")
	}
	if result.Ran || result.SkippedReason != "locked" {
		t.Fatalf("result = %#v, want locked skip", result)
	}
}
