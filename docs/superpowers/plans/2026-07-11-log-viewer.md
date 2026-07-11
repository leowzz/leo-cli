# Log Viewer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a secure, temporary browser-based log workspace started by `leo log`, with recursive discovery, streamed historical search, and live follow.

**Architecture:** Keep project/root resolution independent from log processing. Build an immutable opaque-ID file catalog at startup, revalidate every file before reading, and expose search/follow through an authenticated in-memory HTTP service whose static assets are embedded in the Go binary.

**Tech Stack:** Go 1.25, Cobra, YAML v3, `net/http`, NDJSON streaming, SSE, vanilla HTML/CSS/JavaScript, `go:embed`.

---

## File Structure

- `internal/config/config.go`: optional `proj` configuration model.
- `internal/config/config_test.go`: backward compatibility and project YAML tests.
- `internal/project/project.go`: nearest-ancestor project matching and configured override handling.
- `internal/project/project_test.go`: nested, custom, ambiguous, missing, and explicit-project cases.
- `internal/logview/catalog.go`: root validation, recursive file discovery, opaque IDs, and boundary revalidation.
- `internal/logview/catalog_test.go`: suffix, hidden, symlink, binary, rotation-name, and escape tests.
- `internal/logview/record.go`: Loguru parsing and raw-line fallback.
- `internal/logview/record_test.go`: structured and unknown record fixtures.
- `internal/logview/search.go`: bounded concurrent scans, filters, cancellation, progress, timeout, and limits.
- `internal/logview/search_test.go`: matching, time, regex, cancellation, limit, and oversized-line tests.
- `internal/logview/follow.go`: tailing, rotation/truncation/replacement events, and bounded delivery.
- `internal/logview/follow_test.go`: append, truncate, rename rotation, replacement, and cancellation tests.
- `internal/logweb/server.go`: one-time bootstrap token, sessions, origin checks, routes, and stream encoders.
- `internal/logweb/server_test.go`: authentication, redirects, API access, expiry, origins, and streamed results.
- `internal/logweb/assets/index.html`: single-screen operational workspace.
- `internal/logweb/assets/app.css`: dense responsive layout and bounded log table styling.
- `internal/logweb/assets/app.js`: catalog rendering, filters, NDJSON search, cancellation, SSE follow, and bounded rows.
- `cmd/log.go`: Cobra flags, startup diagnostics, listener lifecycle, and signal-aware shutdown.
- `cmd/log_test.go`: flag defaults, startup resolution, and command errors.
- `README.md`, `README.zh.md`: configuration, SSH/server workflow, and security notes.

## Task 1: Project Configuration

- [ ] Add a failing config test loading `proj.mindcraft.logs` and `proj.mc.match` while preserving a config with no `proj` section.
- [ ] Run `go test ./internal/config` and confirm it fails because `Projects` is undefined.
- [ ] Add `Projects map[string]ProjectConfig \`yaml:"proj"\`` and `ProjectConfig{Match string, Logs []string}` without changing generated defaults.
- [ ] Run `go test ./internal/config` and confirm all tests pass.
- [ ] Commit with `git commit -m "feat: add project log configuration"`.

## Task 2: Project Identification

- [ ] Write table-driven failing tests for nested working directories, default key matching, custom `match`, nearest ancestor, ambiguity, no match, and explicit project selection.
- [ ] Run `go test ./internal/project` and confirm the package/function is missing.
- [ ] Implement `Resolve(cwd, requested string, projects map[string]config.ProjectConfig) (Selection, error)` where `Selection` contains `Name`, `Root`, and `Config`.
- [ ] Reject an unknown explicit project and require even explicit selection to find a matching ancestor.
- [ ] Run `go test ./internal/project` and confirm all cases pass.
- [ ] Commit with `git commit -m "feat: resolve configured project from cwd"`.

## Task 3: Safe File Catalog

- [ ] Write failing tests that validate relative/absolute roots and recursively include `.log`, `.log.*`, `.out`, `.err`, `.tsv`, `.jsonl`, and `.ndjson`.
- [ ] Add failing cases for missing roots, files used as roots, hidden files/directories, symlinks, compressed names, binary prefixes, and an all-invalid-root error.
- [ ] Implement `BuildCatalog(projectRoot string, configured []string) (*Catalog, []string, error)` with canonical allowed roots and stable opaque SHA-256-derived IDs.
- [ ] Implement `Catalog.Resolve(id string) (File, error)` that re-lstats, rejects symlinks/non-regular files, resolves the canonical path, and checks it remains beneath its recorded root.
- [ ] Run `go test ./internal/logview -run 'Catalog|Discover'` and confirm all tests pass.
- [ ] Commit with `git commit -m "feat: discover logs within configured roots"`.

## Task 4: Record Parsing

- [ ] Write failing fixtures for `timestamp | level | search_id | user_id | source - message`, whitespace, empty IDs, invalid timestamps, and unknown raw lines.
- [ ] Implement `ParseLine(fileID, fileName string, offset int64, line []byte) Record`, preserving raw text and replacing invalid UTF-8.
- [ ] Populate structured fields only when the separators match; keep timestamp parse failure visible without dropping the record.
- [ ] Run `go test ./internal/logview -run Parse` and confirm all tests pass.
- [ ] Commit with `git commit -m "feat: parse structured log records"`.

## Task 5: Bounded Historical Search

- [ ] Write failing tests for candidate selection by file mtime, literal include-all/exclude-any, case sensitivity, field filters, selected IDs, and explicit regex errors.
- [ ] Add failing tests proving context cancellation returns promptly, result limits stop scans, per-line limits mark truncation, and one unreadable file emits a warning instead of aborting.
- [ ] Implement `Searcher.Search(ctx, Query, func(Event) error) error` using a bounded worker pool, scanner buffers, result/time limits, and `result`, `progress`, `warning`, and `done` events.
- [ ] Ensure unparseable timestamps survive when the containing file is a time candidate, while parsed timestamps obey the exact range.
- [ ] Run `go test ./internal/logview -run Search` and confirm all tests pass, including `-race`.
- [ ] Commit with `git commit -m "feat: stream bounded log searches"`.

## Task 6: Live Follow

- [ ] Write failing temporary-file tests for starting near EOF, appends, truncate, deletion, replacement, rename rotation, and context cancellation.
- [ ] Implement `Follower.Follow(ctx, []string, func(FollowEvent) error) error` with polling, file identity/offset tracking, old-file drain on rename, and visible system events.
- [ ] Bound each read and callback delivery so a cancelled or slow client cannot accumulate unbounded records.
- [ ] Run `go test ./internal/logview -run Follow -race` and confirm all tests pass.
- [ ] Commit with `git commit -m "feat: follow active logs with rotation handling"`.

## Task 7: Authenticated HTTP Service

- [ ] Write failing `httptest` cases for token exchange, clean redirect, one-time token rejection, cookie-only APIs, session expiry, same-origin validation, and unknown file IDs.
- [ ] Implement `logweb.New(catalog, options) *Server`, generating cryptographically random bootstrap/session tokens kept only in memory.
- [ ] Add `GET /bootstrap?token=...`, `GET /api/files`, `POST /api/search` (NDJSON), `GET /api/follow` (SSE), and embedded static asset routes.
- [ ] Apply `HttpOnly`, `SameSite=Strict`, bounded expiry, no CORS, security headers, and origin checks for search/follow setup.
- [ ] Run `go test ./internal/logweb` and confirm all tests pass.
- [ ] Commit with `git commit -m "feat: serve authenticated log APIs"`.

## Task 8: Browser Workspace

- [ ] Create a semantic single-screen page with file tree, level shortcuts, time/search/structured filters, progress, cancel, log table, and follow footer.
- [ ] Implement incremental NDJSON consumption, search cancellation, field-click filters, row expansion, and visible errors.
- [ ] Implement SSE follow, pause-on-scroll without pausing receipt, bounded DOM/record buffers, waiting/discarded counters, and jump-to-latest.
- [ ] Add a browser smoke fixture/server and verify desktop and mobile screenshots using Playwright; correct overflow, overlap, keyboard focus, and empty/loading/error states.
- [ ] Run `go test ./internal/logweb` so embedded-asset and route tests confirm the binary contains the workspace.
- [ ] Commit with `git commit -m "feat: add embedded log workspace"`.

## Task 9: `leo log` Command

- [ ] Write failing command tests for `Use`, default host/port, missing projects, unknown `--project`, invalid roots, and startup output.
- [ ] Implement `cmd/log.go` with `--project`, `--host` (default `127.0.0.1`), and `--port` (default `0`).
- [ ] Resolve the project from cwd, build the catalog, print project/root/roots/warnings/bootstrap URL, and serve until command context cancellation.
- [ ] Use a pre-bound `net.Listener`, graceful HTTP shutdown, and no automatic browser launch.
- [ ] Run `go test ./cmd` and `go run . log --help`.
- [ ] Commit with `git commit -m "feat: add leo log command"`.

## Task 10: Documentation and End-to-End Verification

- [ ] Document `proj` configuration, relative-root behavior, flags, SSH/manual URL flow, loopback default, trusted-network limitation, and absence of persistent storage in both READMEs.
- [ ] Run `gofmt -w cmd internal` and `go vet ./...`.
- [ ] Run `go test ./...` and `go test -race ./internal/project ./internal/logview ./internal/logweb ./cmd`.
- [ ] Build with `make build`, create a temporary configured project/log tree, start `bin/leo log`, exchange the token, list files, run a streamed search, append a live record, and stop with SIGINT.
- [ ] Generate a sparse/mocked large-log test fixture and verify first-result streaming and bounded allocations without committing generated data.
- [ ] Review the final diff for secrets, arbitrary-path APIs, unintended persistence, generated artifacts, and unrelated changes.
- [ ] Commit with `git commit -m "docs: document log viewer workflow"`.
