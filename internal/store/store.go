package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Repo struct {
	Path              string
	Name              string
	CurrentBranch     string
	LastCommitAt      time.Time
	LastGitActivityAt time.Time
	LastIndexedAt     time.Time
}

type Store struct {
	db *sql.DB
}

func DefaultPath() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "leo-cli", "leo-cli.sqlite3"), nil
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.init(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) init() error {
	statements := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA busy_timeout = 5000;",
		`CREATE TABLE IF NOT EXISTS repos (
			path TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			current_branch TEXT NOT NULL DEFAULT '',
			last_commit_at INTEGER NOT NULL DEFAULT 0,
			last_git_activity_at INTEGER NOT NULL,
			last_indexed_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_repos_last_git_activity_at
			ON repos(last_git_activity_at DESC);`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	if err := s.ensureColumn("repos", "current_branch", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("repos", "last_commit_at", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureColumn(table, column, definition string) error {
	rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = s.db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
	return err
}

func (s *Store) UpsertRepos(ctx context.Context, repos []Repo) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO repos (
		path, name, current_branch, last_commit_at, last_git_activity_at, last_indexed_at
	) VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(path) DO UPDATE SET
		name = excluded.name,
		current_branch = excluded.current_branch,
		last_commit_at = excluded.last_commit_at,
		last_git_activity_at = excluded.last_git_activity_at,
		last_indexed_at = excluded.last_indexed_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, repo := range repos {
		if _, err := stmt.ExecContext(
			ctx,
			repo.Path,
			repo.Name,
			repo.CurrentBranch,
			unixOrZero(repo.LastCommitAt),
			repo.LastGitActivityAt.Unix(),
			repo.LastIndexedAt.Unix(),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) UpdateRepoMetadata(ctx context.Context, repos []Repo) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE repos SET
		current_branch = ?,
		last_commit_at = ?,
		last_git_activity_at = ?,
		last_indexed_at = ?
		WHERE path = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, repo := range repos {
		if _, err := stmt.ExecContext(
			ctx,
			repo.CurrentBranch,
			unixOrZero(repo.LastCommitAt),
			repo.LastGitActivityAt.Unix(),
			repo.LastIndexedAt.Unix(),
			repo.Path,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) ListRepos(ctx context.Context) ([]Repo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path, name, current_branch, last_commit_at, last_git_activity_at, last_indexed_at
		FROM repos
		ORDER BY last_git_activity_at DESC, path ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var repo Repo
		var lastCommitAt int64
		var lastGitActivityAt int64
		var lastIndexedAt int64
		if err := rows.Scan(&repo.Path, &repo.Name, &repo.CurrentBranch, &lastCommitAt, &lastGitActivityAt, &lastIndexedAt); err != nil {
			return nil, err
		}
		repo.LastCommitAt = timeFromUnixOrZero(lastCommitAt)
		repo.LastGitActivityAt = time.Unix(lastGitActivityAt, 0)
		repo.LastIndexedAt = time.Unix(lastIndexedAt, 0)
		repos = append(repos, repo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return repos, nil
}

func (s *Store) RepoCount(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM repos").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) LatestIndexedAt(ctx context.Context) (time.Time, error) {
	var latest sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT MAX(last_indexed_at) FROM repos").Scan(&latest); err != nil {
		return time.Time{}, err
	}
	if !latest.Valid || latest.Int64 == 0 {
		return time.Time{}, nil
	}
	return time.Unix(latest.Int64, 0), nil
}

func unixOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}

func timeFromUnixOrZero(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(value, 0)
}
