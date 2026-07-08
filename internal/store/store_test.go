package store

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestOpenAppliesWALAndCreatesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "leo-cli.sqlite3")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	var journalMode string
	if err := store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode scan error = %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	var tableName string
	err = store.db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'repos'").Scan(&tableName)
	if err != nil {
		t.Fatalf("schema table lookup error = %v", err)
	}
	if tableName != "repos" {
		t.Fatalf("table = %q, want repos", tableName)
	}
}

func TestUpsertReposAndListReposOrdersNewestFirst(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "leo-cli.sqlite3")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	older := time.Unix(100, 0)
	newer := time.Unix(200, 0)
	indexed := time.Unix(300, 0)

	err = store.UpsertRepos(ctx, []Repo{
		{Path: "/tmp/old", Name: "old", CurrentBranch: "main", LastCommitAt: older, LastGitActivityAt: older, LastIndexedAt: indexed},
		{Path: "/tmp/new", Name: "new", CurrentBranch: "feature", LastCommitAt: newer, LastGitActivityAt: newer, LastIndexedAt: indexed},
	})
	if err != nil {
		t.Fatalf("UpsertRepos() error = %v", err)
	}

	err = store.UpsertRepos(ctx, []Repo{
		{Path: "/tmp/old", Name: "old-renamed", CurrentBranch: "develop", LastCommitAt: time.Unix(225, 0), LastGitActivityAt: time.Unix(250, 0), LastIndexedAt: time.Unix(400, 0)},
	})
	if err != nil {
		t.Fatalf("second UpsertRepos() error = %v", err)
	}

	repos, err := store.ListRepos(ctx)
	if err != nil {
		t.Fatalf("ListRepos() error = %v", err)
	}

	gotPaths := []string{repos[0].Path, repos[1].Path}
	wantPaths := []string{"/tmp/old", "/tmp/new"}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("paths = %#v, want %#v", gotPaths, wantPaths)
	}
	if repos[0].Name != "old-renamed" {
		t.Fatalf("updated name = %q, want old-renamed", repos[0].Name)
	}
	if repos[0].LastGitActivityAt.Unix() != 250 {
		t.Fatalf("updated activity = %d, want 250", repos[0].LastGitActivityAt.Unix())
	}
	if repos[0].CurrentBranch != "develop" {
		t.Fatalf("updated branch = %q, want develop", repos[0].CurrentBranch)
	}
	if repos[0].LastCommitAt.IsZero() || repos[0].LastCommitAt.Unix() != 225 {
		t.Fatalf("updated commit time = %v, want unix 225", repos[0].LastCommitAt)
	}
}

func TestRepoCountAndLatestIndexedAt(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "leo-cli.sqlite3")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	count, err := store.RepoCount(ctx)
	if err != nil {
		t.Fatalf("RepoCount() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
	latest, err := store.LatestIndexedAt(ctx)
	if err != nil {
		t.Fatalf("LatestIndexedAt() error = %v", err)
	}
	if !latest.IsZero() {
		t.Fatalf("latest = %v, want zero", latest)
	}

	err = store.UpsertRepos(ctx, []Repo{
		{Path: "/tmp/old", Name: "old", LastGitActivityAt: time.Unix(100, 0), LastIndexedAt: time.Unix(200, 0)},
		{Path: "/tmp/new", Name: "new", LastGitActivityAt: time.Unix(150, 0), LastIndexedAt: time.Unix(300, 0)},
	})
	if err != nil {
		t.Fatalf("UpsertRepos() error = %v", err)
	}

	count, err = store.RepoCount(ctx)
	if err != nil {
		t.Fatalf("RepoCount() after upsert error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	latest, err = store.LatestIndexedAt(ctx)
	if err != nil {
		t.Fatalf("LatestIndexedAt() after upsert error = %v", err)
	}
	if latest.Unix() != 300 {
		t.Fatalf("latest = %d, want 300", latest.Unix())
	}
}

func TestUpdateRepoMetadataOnlyChangesExistingRows(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "leo-cli.sqlite3")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	err = store.UpsertRepos(ctx, []Repo{
		{
			Path:              "/tmp/repo",
			Name:              "repo",
			CurrentBranch:     "main",
			LastCommitAt:      time.Unix(100, 0),
			LastGitActivityAt: time.Unix(200, 0),
			LastIndexedAt:     time.Unix(300, 0),
		},
	})
	if err != nil {
		t.Fatalf("UpsertRepos() error = %v", err)
	}

	err = store.UpdateRepoMetadata(ctx, []Repo{
		{
			Path:              "/tmp/repo",
			CurrentBranch:     "feature",
			LastCommitAt:      time.Unix(400, 0),
			LastGitActivityAt: time.Unix(500, 0),
			LastIndexedAt:     time.Unix(600, 0),
		},
		{
			Path:              "/tmp/missing",
			CurrentBranch:     "ignored",
			LastCommitAt:      time.Unix(700, 0),
			LastGitActivityAt: time.Unix(800, 0),
			LastIndexedAt:     time.Unix(900, 0),
		},
	})
	if err != nil {
		t.Fatalf("UpdateRepoMetadata() error = %v", err)
	}

	repos, err := store.ListRepos(ctx)
	if err != nil {
		t.Fatalf("ListRepos() error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("repos len = %d, want 1", len(repos))
	}
	repo := repos[0]
	if repo.Name != "repo" {
		t.Fatalf("name = %q, want repo", repo.Name)
	}
	if repo.CurrentBranch != "feature" {
		t.Fatalf("branch = %q, want feature", repo.CurrentBranch)
	}
	if repo.LastCommitAt.Unix() != 400 {
		t.Fatalf("commit = %d, want 400", repo.LastCommitAt.Unix())
	}
	if repo.LastGitActivityAt.Unix() != 500 {
		t.Fatalf("activity = %d, want 500", repo.LastGitActivityAt.Unix())
	}
	if repo.LastIndexedAt.Unix() != 600 {
		t.Fatalf("indexed = %d, want 600", repo.LastIndexedAt.Unix())
	}
}
