# leo-cli Repo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first working Go version of `leo-cli repo` and `leo-cli repo reindex`.

**Architecture:** The CLI uses Cobra for command routing, a config package for YAML defaults and path expansion, a store package for SQLite/WAL access, a repoindex package for filesystem scanning, and a repoui package for Bubble Tea interaction. The first version stores repo rows by absolute path and lists them by last Git activity descending.

**Tech Stack:** Go, Cobra, Bubble Tea, Bubbles, Lipgloss, YAML v3, modernc SQLite.

---

## File Structure

- `go.mod`: Go module and dependencies.
- `main.go`: program entrypoint.
- `cmd/root.go`: root command and shared config/store setup helpers.
- `cmd/repo.go`: `repo` and `repo reindex` commands.
- `internal/config/config.go`: config path resolution, default creation, YAML load, and path expansion.
- `internal/config/config_test.go`: config behavior tests.
- `internal/store/store.go`: SQLite open, WAL pragmas, schema, upsert, and list.
- `internal/store/store_test.go`: store tests.
- `internal/repoindex/repoindex.go`: Git repo scanning and activity timestamp extraction.
- `internal/repoindex/repoindex_test.go`: scanner tests.
- `internal/repoui/repoui.go`: Bubble Tea repo list UI.

## Task 1: Go Module Skeleton

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

- [ ] **Step 1: Initialize module**

Run:

```bash
go mod init github.com/leo/leo-cli
```

Expected: `go.mod` exists with module path `github.com/leo/leo-cli`.

- [ ] **Step 2: Add root command**

Create `main.go`:

```go
package main

import "github.com/leo/leo-cli/cmd"

func main() {
	cmd.Execute()
}
```

Create `cmd/root.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "leo-cli",
	Short: "Personal command-line tools",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Add dependency and verify**

Run:

```bash
go get github.com/spf13/cobra@latest
go test ./...
```

Expected: `go test ./...` passes.

## Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write config tests**

Create `internal/config/config_test.go` with tests for default creation, YAML loading, `~` expansion, and environment expansion.

- [ ] **Step 2: Implement config package**

Create `internal/config/config.go` with:

```go
type Config struct {
	Repo RepoConfig `yaml:"repo"`
}

type RepoConfig struct {
	Roots []string `yaml:"roots"`
}
```

Implement:

- `DefaultPath() (string, error)`
- `Ensure(path string) error`
- `Load(path string) (Config, error)`
- `ExpandPath(path string) (string, error)`
- `ExpandedRepoRoots(cfg Config) ([]string, error)`

- [ ] **Step 3: Add YAML dependency and verify**

Run:

```bash
go get gopkg.in/yaml.v3@latest
go test ./internal/config
```

Expected: config tests pass.

## Task 3: SQLite Store

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`

- [ ] **Step 1: Write store tests**

Create tests that open a temp DB, migrate schema, upsert repos, and assert list order is newest first.

- [ ] **Step 2: Implement store**

Implement:

```go
type Repo struct {
	Path              string
	Name              string
	LastGitActivityAt time.Time
	LastIndexedAt     time.Time
}
```

Functions:

- `DefaultPath() (string, error)`
- `Open(path string) (*Store, error)`
- `Close() error`
- `UpsertRepos(ctx context.Context, repos []Repo) error`
- `ListRepos(ctx context.Context) ([]Repo, error)`

On open, apply WAL pragmas and create schema.

- [ ] **Step 3: Add SQLite dependency and verify**

Run:

```bash
go get modernc.org/sqlite@latest
go test ./internal/store
```

Expected: store tests pass.

## Task 4: Repo Scanner

**Files:**
- Create: `internal/repoindex/repoindex.go`
- Create: `internal/repoindex/repoindex_test.go`

- [ ] **Step 1: Write scanner tests**

Create temp directory tests for:

- normal repo with `.git/logs/HEAD`
- worktree-style `.git` file pointing to a real gitdir
- missing root reporting

- [ ] **Step 2: Implement scanner**

Implement:

```go
type ScanResult struct {
	Repos    []store.Repo
	Warnings []string
}

func ScanRoots(roots []string, now time.Time) ScanResult
```

Use `filepath.WalkDir`, detect `.git`, resolve gitdir for `.git` file content shaped like `gitdir: <path>`, extract last timestamp from `logs/HEAD`, and fall back to metadata mtimes.

- [ ] **Step 3: Verify**

Run:

```bash
go test ./internal/repoindex
```

Expected: scanner tests pass.

## Task 5: Repo Commands

**Files:**
- Modify: `cmd/root.go`
- Create: `cmd/repo.go`

- [ ] **Step 1: Wire config and store helpers**

Add helper functions in `cmd/root.go` for loading config and opening the store with default paths.

- [ ] **Step 2: Implement `repo reindex`**

Create `cmd/repo.go` with:

- `repoCmd`
- `repoReindexCmd`
- config load
- root expansion
- scan
- store upsert
- warning output
- count summary

- [ ] **Step 3: Add temporary non-interactive `repo` list command**

Before TUI implementation, make `leo-cli repo` list repo paths in sorted order so command integration can be tested.

- [ ] **Step 4: Verify**

Run:

```bash
go test ./...
go run . repo reindex
go run . repo
```

Expected: tests pass, reindex creates default config and DB, and repo command prints indexed repos or an empty-index suggestion.

## Task 6: Bubble Tea UI

**Files:**
- Create: `internal/repoui/repoui.go`
- Modify: `cmd/repo.go`

- [ ] **Step 1: Implement repo list UI**

Create a Bubble Tea model with:

- list items from `store.Repo`
- filtering by name/path through `bubbles/list`
- Enter selection
- Esc/Ctrl-C exit

Expose:

```go
func Run(repos []store.Repo) (string, bool, error)
```

The returned bool is true when a repo is selected.

- [ ] **Step 2: Wire `leo-cli repo` to UI**

In `cmd/repo.go`, load repos and call `repoui.Run`. If selected, print the selected path.

- [ ] **Step 3: Add UI dependencies and verify**

Run:

```bash
go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest github.com/charmbracelet/lipgloss@latest
go test ./...
go run . repo
```

Expected: tests pass and the interactive repo list opens when indexed repos exist.

## Task 7: Final Verification

**Files:**
- Modify only if verification exposes defects.

- [ ] **Step 1: Format**

Run:

```bash
gofmt -w main.go cmd internal
```

- [ ] **Step 2: Test**

Run:

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 3: Smoke test**

Run:

```bash
go run . repo reindex
go run . repo
```

Expected: `repo reindex` scans configured roots and `repo` opens a usable interactive list or prints a clear empty-index message.
