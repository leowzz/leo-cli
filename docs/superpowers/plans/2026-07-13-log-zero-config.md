# Zero-Configuration Log Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `leo log` launch from an unconfigured project by safely discovering log directories, with repeatable `--logs` overrides and actionable failures.

**Architecture:** Configured project resolution remains authoritative and exposes a typed no-match result for safe fallback. Ad-hoc root selection lives in `internal/project`, bounded log-directory discovery lives in a focused command helper, and `cmd/log.go` selects configured, explicit-root, or automatic mode before using the unchanged log catalog/server.

**Tech Stack:** Go, Cobra, filesystem APIs, existing `internal/logview` catalog and Go test suite

---

## File Structure

- `internal/project/project.go`: expose `ErrNoMatch` and nearest-Git-root fallback.
- `internal/project/project_test.go`: prove typed no-match and Git/non-Git root selection.
- `cmd/log_discovery.go`: implement conventional and bounded log-directory discovery plus explicit path expansion.
- `cmd/log_discovery_test.go`: cover discovery priority, skips, depth, sorting, and expansion.
- `cmd/log.go`: register `--logs`, enforce mode precedence, reject empty catalogs, and label auto mode.
- `cmd/log_test.go`: cover configured/automatic/explicit runtime policy and friendly errors.
- `README.md`: document first-run `leo log` and `--logs`.

### Task 1: Typed Project Fallback and Ad-Hoc Root

**Files:**
- Modify: `internal/project/project.go`
- Modify: `internal/project/project_test.go`

- [ ] **Step 1: Write failing project tests**

Add tests that assert implicit no-match wraps a sentinel, while ambiguous and explicit-selection errors do not. Add Git-root tests for a nested directory, a `.git` file, and a non-Git directory.

```go
func TestResolveReturnsTypedNoMatch(t *testing.T) {
    _, err := Resolve(t.TempDir(), "", map[string]config.ProjectConfig{"other": {}})
    if !errors.Is(err, ErrNoMatch) {
        t.Fatalf("Resolve() error = %v, want ErrNoMatch", err)
    }
}

func TestFindRootUsesNearestGitMarker(t *testing.T) {
    root := t.TempDir()
    nested := filepath.Join(root, "service", "pkg")
    os.MkdirAll(nested, 0o755)
    os.Mkdir(filepath.Join(root, ".git"), 0o755)
    got, err := FindRoot(nested)
    if err != nil || got != root {
        t.Fatalf("FindRoot() = %q, %v, want %q", got, err, root)
    }
}

func TestFindRootFallsBackToCurrentDirectory(t *testing.T) {
    cwd := t.TempDir()
    got, err := FindRoot(cwd)
    if err != nil || got != cwd {
        t.Fatalf("FindRoot() = %q, %v, want %q", got, err, cwd)
    }
}
```

- [ ] **Step 2: Run project tests and verify RED**

```bash
go test ./internal/project -run 'Test(ResolveReturnsTypedNoMatch|FindRoot)' -count=1
```

Expected: FAIL because `ErrNoMatch` and `FindRoot` do not exist.

- [ ] **Step 3: Add the sentinel and preserve human-readable errors**

```go
var ErrNoMatch = errors.New("no configured project match")
```

Wrap only the final implicit no-match return:

```go
return Selection{}, fmt.Errorf(
    "%w: no configured project matches %q; configured projects: %s; use --project",
    ErrNoMatch, cleanCWD, projectNames(projects),
)
```

Unknown explicit projects, explicit roots outside the selected project, and
ambiguity remain ordinary errors.

- [ ] **Step 4: Implement nearest Git root**

Add:

```go
func FindRoot(cwd string) (string, error) {
    current, err := filepath.Abs(cwd)
    if err != nil {
        return "", fmt.Errorf("resolve project root: %w", err)
    }
    current = filepath.Clean(current)
    for dir := current; ; {
        info, statErr := os.Lstat(filepath.Join(dir, ".git"))
        if statErr == nil && (info.IsDir() || info.Mode().IsRegular()) {
            return dir, nil
        }
        if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
            return "", fmt.Errorf("inspect Git root %q: %w", dir, statErr)
        }
        parent := filepath.Dir(dir)
        if parent == dir {
            return current, nil
        }
        dir = parent
    }
}
```

- [ ] **Step 5: Run package tests and commit**

```bash
gofmt -w internal/project
go test ./internal/project -count=1
git diff --check
git add internal/project/project.go internal/project/project_test.go
git commit -m "feat: support ad-hoc project roots"
```

Expected: all commands PASS.

### Task 2: Bounded Log Directory Discovery

**Files:**
- Create: `cmd/log_discovery.go`
- Create: `cmd/log_discovery_test.go`

- [ ] **Step 1: Write failing discovery tests**

Cover all conventional directories, conventional-path priority over traversal,
case-insensitive bounded matches, maximum depth four, skipped hidden/build
trees, deterministic sorting, accepted-root descent stopping, and explicit
path expansion relative to invocation CWD.

Use this table in the tests:

```go
var expectedConventionalLogRoots = []string{
    "runtime/logs",
    "logs",
    "log",
    "var/log",
    "storage/logs",
}
```

Examples:

```go
func TestDiscoverLogRootsPrefersAllConventionalDirectories(t *testing.T) {
    root := t.TempDir()
    for _, relative := range expectedConventionalLogRoots {
        os.MkdirAll(filepath.Join(root, relative), 0o755)
    }
    os.MkdirAll(filepath.Join(root, "service", "logs"), 0o755)

    got, warnings := discoverLogRoots(root)
    // Assert no warnings, all five conventional absolute paths sorted,
    // and no service/logs traversal result.
}

func TestDiscoverLogRootsBoundsAndSkipsTraversal(t *testing.T) {
    root := t.TempDir()
    // Create app/Logs at depth 2, a/b/c/logs at depth 4,
    // a/b/c/d/logs at depth 5, and node_modules/pkg/logs.
    // Assert only the first two are returned.
}
```

Add `expandExplicitLogRoots(cwd, values)` tests for relative, absolute,
`~/...`, environment-expanded, duplicate, and empty values.

- [ ] **Step 2: Run focused tests and verify RED**

```bash
go test ./cmd -run 'Test(DiscoverLogRoots|ExpandExplicitLogRoots)' -count=1
```

Expected: FAIL because the new helpers do not exist.

- [ ] **Step 3: Implement conventional discovery**

Create constants:

```go
var conventionalLogRoots = []string{
    "runtime/logs",
    "logs",
    "log",
    "var/log",
    "storage/logs",
}

var skippedLogDiscoveryDirs = map[string]struct{}{
    ".git": {}, "node_modules": {}, "vendor": {}, "dist": {},
    "build": {}, "target": {}, ".venv": {}, "venv": {},
    "__pycache__": {},
}
```

`discoverLogRoots(root) ([]string, []string)` checks every conventional path,
adds directories, records non-not-exist stat errors as warnings, cleans and
sorts results, and returns immediately when at least one conventional directory
exists.

- [ ] **Step 4: Implement bounded traversal**

When conventional discovery is empty, use `filepath.WalkDir`.

- Compute depth from `filepath.Rel(root, path)`.
- Skip hidden or configured skipped directories.
- Skip every directory deeper than four components.
- Match basename `log` or `logs` with `strings.EqualFold`.
- Add the absolute cleaned directory and return `filepath.SkipDir`.
- Convert walk/stat failures to warning strings without discarding other roots.
- Deduplicate and sort before returning.

Do not follow directory symlinks; `WalkDir` already reports them as entries,
and matching requires `entry.IsDir()`.

- [ ] **Step 5: Implement explicit path expansion**

`expandExplicitLogRoots(cwd string, values []string) ([]string, error)`:

1. reject empty/whitespace values;
2. call `os.ExpandEnv`;
3. use `config.ExpandPath` for `~` paths;
4. join other relative paths to `cwd`;
5. convert to cleaned absolute paths;
6. deduplicate and sort.

The catalog remains responsible for existence, canonicalization, readability,
and directory safety.

- [ ] **Step 6: Run tests and commit**

```bash
gofmt -w cmd/log_discovery.go cmd/log_discovery_test.go
go test ./cmd -run 'Test(DiscoverLogRoots|ExpandExplicitLogRoots)' -count=1
go test ./cmd -count=1
git diff --check
git add cmd/log_discovery.go cmd/log_discovery_test.go
git commit -m "feat: discover project log directories"
```

Expected: all commands PASS.

### Task 3: Runtime Policy, `--logs`, and Friendly Errors

**Files:**
- Modify: `cmd/log.go`
- Modify: `cmd/log_test.go`
- Modify: `README.md`

- [ ] **Step 1: Write failing runtime-policy tests**

Extend default flag coverage to require a repeatable `logs` flag. Replace the
old missing-configuration rejection test with:

- empty config plus conventional logs succeeds in auto mode;
- unmatched configured projects fall back to auto mode;
- matching configured project wins over auto roots;
- explicit unknown/mismatched `--project` remains strict;
- explicit `--logs` uses only supplied roots;
- `--project` plus `--logs` is rejected;
- an empty auto catalog reports root, discovery summary, executable `--logs`
  command, and minimal `proj:` YAML;
- explicit empty/unusable `--logs` names the supplied roots;
- startup output appends `(auto)` only for ad-hoc mode.

The new signature is:

```go
func prepareLogRuntime(
    cfg config.Config,
    cwd string,
    requestedProject string,
    explicitLogRoots []string,
) (logRuntime, []string, error)
```

- [ ] **Step 2: Run focused command tests and verify RED**

```bash
go test ./cmd -run 'Test(LogCommandDefaults|PrepareLogRuntime|PrintLogStartup)' -count=1
```

Expected: FAIL from the old signature, missing flag, and missing auto behavior.

- [ ] **Step 3: Add flag and runtime mode**

Add:

```go
var logRoots []string

type logRuntime struct {
    project   project.Selection
    catalog   *logview.Catalog
    automatic bool
}
```

Register:

```go
logCmd.Flags().StringArrayVar(
    &logRoots, "logs", nil,
    "Log directory (repeat for multiple roots)",
)
logCmd.MarkFlagsMutuallyExclusive("project", "logs")
```

Pass `logRoots` into `runLogServer` and `prepareLogRuntime`. Keep an
explicit mutual-exclusion check inside `prepareLogRuntime` so unit callers
receive the same policy even without Cobra validation.

- [ ] **Step 4: Implement configured/explicit/automatic selection**

Use this policy:

```go
if requestedProject != "" {
    return prepareConfiguredLogRuntime(cfg, cwd, requestedProject)
}
if len(explicitLogRoots) > 0 {
    return prepareAdHocLogRuntime(cwd, explicitLogRoots, true)
}

selection, err := project.Resolve(cwd, "", cfg.Projects)
if err == nil {
    return buildConfiguredLogRuntime(selection)
}
if !errors.Is(err, project.ErrNoMatch) {
    return logRuntime{}, nil, err
}
return prepareAdHocLogRuntime(cwd, nil, false)
```

The actual helpers may share catalog construction, but configured errors must
not enter auto mode.

For ad-hoc mode:

- call `project.FindRoot(cwd)`;
- explicit mode calls `expandExplicitLogRoots(cwd, values)`;
- automatic mode calls `discoverLogRoots(root)`;
- build a `project.Selection` with basename name, detected root, and selected
  roots;
- call `logview.BuildCatalog`;
- reject zero roots and zero files with mode-specific friendly errors;
- preserve discovery/catalog warnings;
- set `automatic: true`.

- [ ] **Step 5: Implement actionable errors and startup label**

Use a focused formatter such as:

```go
func autoLogDiscoveryError(root string, warnings []string) error {
    return fmt.Errorf(
        "no log files found for %s\n"+
            "tried common log directories and log/logs folders up to depth 4\n"+
            "run: leo log --logs ./path/to/logs\n"+
            "or add:\n  proj:\n    %s:\n      logs:\n        - runtime/logs%s",
        root,
        filepath.Base(root),
        formatLogWarnings(warnings),
    )
}
```

Ensure warnings are line-oriented and deterministic. Explicit-root errors
suggest correcting `--logs` and include the supplied values.

Update `printLogStartup`:

```go
name := runtime.project.Name
if runtime.automatic {
    name += " (auto)"
}
fmt.Fprintf(stdout, "Project: %s\n", name)
```

- [ ] **Step 6: Update README**

Lead the Log Viewer section with zero-config usage:

```bash
cd /path/to/project
leo log
```

Explain conventional/bounded discovery, then show:

```bash
leo log --logs ./custom/logs
leo log --logs ./api/logs --logs ./worker/logs
```

Keep configured `proj:` as the persistent customization path and document
that `--project` remains strict.

- [ ] **Step 7: Run package tests and commit**

```bash
gofmt -w cmd/log.go cmd/log_test.go
go test ./cmd -count=1
go test ./internal/project -count=1
git diff --check
git add cmd/log.go cmd/log_test.go README.md
git commit -m "feat: run log viewer without project config"
```

Expected: all commands PASS.

### Task 4: Integration and Repository Verification

**Files:**
- Modify only if verification exposes a tested defect.

- [ ] **Step 1: Exercise a real zero-config project**

Create an isolated Git project with no `proj:` configuration and a current
`runtime/logs/error.log`. Build the current binary and start `leo log` from
a nested directory without `--project` or `--logs`.

Verify startup prints `Project: <basename> (auto)`, the detected Git root, and
the discovered log root. Open the bootstrap URL, list files, run a historical
search, observe Follow, append a record, and confirm it streams.

Stop with SIGINT and confirm clean shutdown.

- [ ] **Step 2: Exercise explicit roots and failure guidance**

Run from a non-Git directory with:

```bash
leo log --logs ./custom/logs
```

Verify the current directory is the root and only the explicit root is used.
Then run in a project without log directories and confirm the error includes
the detected root, bounded-search summary, executable `--logs` command, and
minimal YAML.

- [ ] **Step 3: Run full verification**

```bash
gofmt -w internal/project cmd
go vet ./...
go test ./... -count=1
go test -race ./internal/project ./internal/logview ./internal/logweb ./cmd -count=1
git diff --check
```

Expected: all commands exit 0 with no race reports.

- [ ] **Step 4: Check repository hygiene**

Run the sensitive-example scan using the repository's split-literal convention.
Remove temporary configs, projects, binaries, browser artifacts, and server
processes. Confirm `git status --short` contains only intended committed work.

- [ ] **Step 5: Commit verification fixes only when necessary**

If integration verification reveals a defect, first add a focused failing
regression test, implement the minimal fix, repeat focused and full checks, and
commit it separately. Do not create an empty commit.
