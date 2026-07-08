# leo-cli Repo Index Design

## Goal

Build the first usable slice of `leo-cli`: a personal command-line tool for browsing commonly used local repositories.

The first release focuses on one closed loop:

- `leo-cli repo reindex` scans configured repository roots and stores repo metadata.
- `leo-cli repo` opens an interactive terminal list backed by that metadata.

Future personal-command surfaces are out of scope for this slice.

## Technology

- Language: Go.
- CLI framework: `cobra`.
- TUI framework: `bubbletea`, `bubbles`, and `lipgloss`.
- Config format: YAML.
- Config path: `~/.config/leo-cli/config.yaml`.
- Data path: `~/.local/share/leo-cli/leo-cli.sqlite3`.
- SQLite driver: `modernc.org/sqlite` to avoid CGO for simpler local builds and binary distribution.

## Configuration

If no config file exists, the CLI creates this default:

```yaml
repo:
  roots:
    - ~/work
```

`repo.roots` is the list of directories recursively scanned by `repo reindex`.

Path expansion supports:

- `~` for the current user's home directory.
- Environment variables accepted by Go's `os.ExpandEnv`.

Missing roots are reported during reindex but do not abort the entire run unless all configured roots are unusable.

## Commands

### `leo-cli repo reindex`

Reads the YAML config, scans each configured root, detects Git repositories, and writes the index into SQLite.

Repository detection:

- A directory containing `.git` is a repository.
- `.git` may be either a directory or a file, so Git worktrees are supported.
- Nested repositories are allowed, but once a repo is found the scanner does not recurse into its `.git` internals.

Indexed fields:

- Absolute path.
- Repository display name, defaulting to the directory basename.
- Last Git activity timestamp.
- Last indexed timestamp.

Last Git activity is computed from the newest available Git metadata signal:

- `.git/logs/HEAD` timestamp when available.
- `.git` metadata mtime as a fallback.
- Repository directory mtime as the final fallback.

Rows are upserted by absolute path. Reindexing does not delete missing repositories in the first version; stale cleanup can be added later with an explicit command or flag.

### `leo-cli repo`

Opens an interactive terminal UI listing indexed repositories.

Behavior:

- Default sort is last Git activity descending.
- Typing filters repositories by display name and path.
- The selected row shows enough path context to distinguish similarly named repos.
- Pressing Enter prints the selected absolute path and exits.
- Escape or Ctrl-C exits without selection.

The first version only browses and selects repositories. It does not run Git commands, open editors, or change directories directly.

## Storage

The database lives at:

```text
~/.local/share/leo-cli/leo-cli.sqlite3
```

On open, the store applies:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
```

Initial schema:

```sql
CREATE TABLE IF NOT EXISTS repos (
  path TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  last_git_activity_at INTEGER NOT NULL,
  last_indexed_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_repos_last_git_activity_at
ON repos(last_git_activity_at DESC);
```

Timestamps are stored as Unix seconds for simple sorting and display.

## Package Boundaries

The implementation should keep these responsibilities separate:

- `cmd`: cobra command wiring and user-facing command behavior.
- `internal/config`: config path resolution, default creation, YAML loading, and path expansion.
- `internal/store`: SQLite connection setup, schema migration, repo upsert, and list queries.
- `internal/repoindex`: filesystem scanning and Git activity extraction.
- `internal/repoui`: Bubble Tea model and terminal list behavior.

## Error Handling

- Config creation errors are fatal because no command can run without configuration.
- Database open or migration errors are fatal.
- Individual unreadable directories during reindex are collected and reported.
- If no repos are found, reindex succeeds with a clear message.
- `leo-cli repo` should suggest running `leo-cli repo reindex` when the index is empty.

## Tests

Focused tests should cover:

- Default config creation and YAML loading.
- `~` and environment-variable path expansion.
- Repository scanning for normal repos and worktree-style `.git` files.
- Git activity timestamp fallback behavior.
- SQLite schema setup and upsert/list ordering.

TUI behavior can be kept light in the first version: unit-test filtering and selection state where practical, and rely on manual smoke testing for terminal rendering.
