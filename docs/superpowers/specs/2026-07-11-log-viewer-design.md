# Leo CLI Log Viewer Design

**Date:** 2026-07-11

## Summary

Add `leo log` to `leo-cli`. The command identifies the current project from the
working directory, loads that project's configured log directories, starts a
temporary HTTP server, and prints a clickable URL for a browser-based log
workspace.

The first version reads existing files in place. It does not collect, copy,
index, or persist log contents. It prioritizes searches over the most recent
hour and supports occasional multi-day scans without building a second log
system.

## Goals

- Start a usable log workspace with one command on a development server.
- Identify the project automatically from the current working directory.
- Restrict all discovery and reads to project-configured log directories.
- Recursively discover common text log files in nested directories.
- Search large log sets on demand and stream results as they are found.
- Follow active files, handle rotation, pause visual scrolling, and resume at
  the latest record.
- Parse demo_01's common Loguru fields for quick filtering while still
  displaying unknown text formats.
- Ship the server and web UI inside the existing Go binary without a separate
  runtime or frontend build toolchain.

## Non-goals

- Log collection or forwarding.
- A persistent full-text index, SQLite log storage, or copied log records.
- Queries across multiple servers.
- Alerts, dashboards, reports, or long-term retention.
- Production-grade public internet exposure.
- Editing or deleting log files.
- Persisting UI state or saved searches in the first version.

## Command

The primary command is:

```bash
leo log
```

Useful overrides are:

```bash
leo log --project demo_01
leo log --host 0.0.0.0 --port 9031
```

The command runs in the foreground until interrupted. It prints the identified
project, project root, allowed log directories, skipped-directory warnings, and
a clickable one-time URL. The HTTP server exits cleanly on `Ctrl-C` and may
exit after an inactivity timeout when no browser stream is connected.

Defaults:

- `--host`: `127.0.0.1`
- `--port`: `0`, letting the OS choose an available port
- initial time range: one hour

Listening on a non-loopback address must be an explicit choice.

## Configuration

The existing `~/.config/leo-cli/config.yaml` gains a short `proj` mapping:

```yaml
proj:
  demo_01:
    logs:
      - runtime/logs
      - /docker-runtime
```

The project key is also the default path match string. A different display key
can override the match string:

```yaml
proj:
  mc:
    match: demo_01
    logs:
      - runtime/logs
```

Proposed Go model:

```go
type Config struct {
    Repo    RepoConfig               `yaml:"repo"`
    Docker  DockerConfig             `yaml:"docker"`
    Time    TimeConfig               `yaml:"time"`
    Projects map[string]ProjectConfig `yaml:"proj"`
}

type ProjectConfig struct {
    Match string   `yaml:"match"`
    Logs  []string `yaml:"logs"`
}
```

The generated default config remains unchanged. `proj` is optional so existing
users and commands keep working without migration.

## Project Identification

1. Resolve the current working directory to a clean absolute path.
2. For every configured project, use `match` when present; otherwise use the
   project's map key.
3. Walk from the current directory toward the filesystem root.
4. The nearest ancestor whose directory name contains a project's match string
   is that project's root.
5. If exactly one project matches at the nearest depth, select it.
6. If no project matches, return an error that lists configured projects and
   suggests `--project`.
7. If multiple projects match at the same nearest depth, return an ambiguity
   error and require `--project`.

`--project` selects the named config entry but does not discard root safety. The
command still walks upward to find that project's matching root. Running the
command from outside the project therefore fails instead of resolving relative
log paths from an unrelated directory.

Relative log paths resolve from the detected project root. Absolute paths stay
absolute. Home and environment expansion reuse `internal/config.ExpandPath`
semantics where applicable.

## Log Directory Boundary

At startup, each configured log directory is cleaned, made absolute, and
checked for existence, directory type, and read permission. Invalid entries
produce warnings. At least one usable directory is required.

The service recursively discovers files under the usable roots. It accepts only
regular text files with these built-in name patterns:

- `*.log`
- `*.log.*`
- `*.out`
- `*.err`
- `*.tsv`
- `*.jsonl`
- `*.ndjson`

Hidden files, symlinks, and compressed files such as `.gz`, `.zip`, `.xz`, and
`.bz2` are skipped. A small prefix check rejects binary content even when the
filename matches.

The browser never sends an arbitrary filesystem path. File discovery assigns
opaque file IDs. Every read resolves the ID through the server-side catalog and
revalidates that the canonical file remains under an allowed root. A replaced
file that becomes a symlink or escapes its root is rejected.

## Server Components

- `cmd/log.go`: Cobra command, flags, configuration loading, process lifetime,
  and terminal output.
- `internal/project`: project matching and project-root resolution.
- `internal/logview`: allowed-root validation, recursive discovery, file IDs,
  parsing, search, following, rotation handling, and resource limits.
- `internal/logweb`: HTTP routes, authentication session, streaming responses,
  and embedded web assets.

The frontend is static HTML, CSS, and JavaScript embedded with `go:embed`. The
first version does not add Node, npm, React, or a separate frontend build step.
The scanner is implemented in Go so the remote server does not need `rg` or
another executable.

## Search Behavior

- The default range is the most recent hour.
- The default selection includes every discovered log file modified within the
  selected time range. The user can narrow the scope from the file tree.
- Candidate selection uses file metadata before content scanning, so the
  recent-hour default does not scan unrelated older files.
- Literal matching is the default, with a separate case-sensitivity toggle.
- Regex matching must be enabled explicitly.
- Level, search ID, user ID, source, selected files, and time range are separate
  filters. The first version does not implement a general query language.
- Multiple include terms must all match. Exclude terms remove matching records.
- The server scans candidate files with a bounded worker pool and bounded read
  buffers.
- Results and progress are streamed as they are found. Disconnecting or
  cancelling the request cancels all scan workers promptly.
- Search stops at a fixed result limit or timeout and reports why it stopped.
- Before a multi-day scan starts, the UI shows the candidate file count and
  total byte size.

The parser recognizes the current demo_01 Loguru layout:

```text
time | level | search_id | user_id | source - message
```

Recognized fields become filterable columns. Lines that do not match a known
format remain searchable raw records. An unparseable timestamp is not silently
discarded by a time filter when its containing file is otherwise a candidate.

Historical search uses a streamed fetch response such as NDJSON so it can be a
POST request with structured filters and still return incremental results.

## Live Follow Behavior

- Following starts near the end of selected active files rather than loading
  the full files.
- The server tracks file identity and byte offset.
- On rename-based rotation, it drains the old file and then opens the new file.
- On truncation, deletion, replacement, or permission failure, it emits a
  visible system record instead of silently stopping.
- Live records use SSE because the data flow is server-to-browser only.
- Scrolling away from the bottom pauses automatic scrolling, not receipt of
  records.
- The browser keeps a bounded record buffer and shows how many new records are
  waiting. Returning to the bottom resumes automatic scrolling.
- When the browser buffer is full, it discards the oldest records and reports
  the discarded count.

## Web Workspace

The page is a dense, single-screen operational tool:

- Left: recursively discovered file tree and level shortcuts.
- Top: time range, text search, literal/regex controls, structured filters,
  scan size, search, and cancel.
- Center: bounded log table with time, level, search ID, user ID, source, and
  message columns.
- Bottom: live-follow state, pause state, buffered-new-record count, and jump to
  latest.

Clicking a search ID, user ID, level, or source adds a structured filter. Long
messages are collapsed and can be expanded without resizing unrelated rows.
Sensitive fields are not automatically persisted.

## Access Control

The server generates a one-time bootstrap token and includes it in the printed
URL. The first successful request exchanges it for a short-lived, HttpOnly,
SameSite session cookie and redirects to a clean URL without the token.

The service does not enable CORS. It validates the request origin for mutating
or streaming setup requests and applies session expiry. Tokens and sessions
remain in memory and are never written to SQLite or config.

Plain HTTP on a non-loopback address is intended only for a trusted development
network. For an untrusted network, the user must keep the default loopback bind
and use SSH port forwarding.

## Resource and Error Limits

The first version uses fixed conservative defaults rather than additional
configuration keys:

- bounded concurrent file scanners;
- bounded result count;
- bounded search duration;
- bounded per-line size;
- bounded browser record count;
- bounded live backlog per client.

Invalid UTF-8 is displayed with replacement characters. Oversized lines are
marked and truncated in the table; a bounded endpoint may retrieve more of that
single line. One unreadable file does not abort a multi-file search, but the UI
shows the warning. Losing every configured root or the authentication session
is fatal and clearly reported.

## Storage

Version one does not use SQLite for the log feature. It does not persist UI
preferences, saved searches, file catalogs, offsets, tokens, sessions, or log
records. Existing repository-index SQLite behavior remains unchanged.

If the feature proves useful, a later version may persist per-project UI state
and explicitly saved queries without persisting log contents.

## Testing

- Configuration loading remains backward compatible when `proj` is absent.
- Project matching covers nested working directories, custom `match`, nearest
  ancestor selection, no match, ambiguous match, and `--project`.
- Root validation covers relative and absolute paths, missing roots, unreadable
  roots, traversal, symlink replacement, hidden files, binary files, compressed
  files, and recursive suffix filtering.
- Parser fixtures cover the demo_01 Loguru format and unknown raw lines.
- Search tests cover time boundaries, include and exclude terms, field filters,
  case sensitivity, regex errors, cancellation, timeout, and result limits.
- Follow tests use temporary files to cover append, truncate, rename rotation,
  replacement, deletion, and slow clients.
- HTTP tests cover bootstrap exchange, clean redirect, expired sessions,
  unauthorized access, origin checks, and cancellation on disconnect.
- Browser tests cover search, click-to-filter, pause on scroll, jump to latest,
  long-line expansion, and bounded record retention.
- A generated data test of at least 1 GB confirms that results stream before a
  full scan finishes and that server memory remains bounded.

## Success Criteria

- From any directory below a configured demo_01 root, `leo log` identifies
  the project and opens a reachable workspace URL.
- The workspace cannot read files outside configured log roots.
- A recent-hour search begins returning matches without first loading or
  indexing the full log set.
- Live follow survives normal append and rotation behavior.
- Long-running searches can be cancelled promptly.
- Server and browser memory do not grow without a configured bound.
- The released `leo` binary remains sufficient to run the feature on a remote
  development server.
