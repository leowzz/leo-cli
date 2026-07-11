# Log Viewer Workspace Context

**Date:** 2026-07-11
**Target repository:** `/Users/leo/Desktop/work/leo-cli`
**Source application inspected:** `/Users/leo/Desktop/work/mindcraft`
**Primary design:** `docs/superpowers/specs/2026-07-11-log-viewer-design.md`

## Purpose

This note preserves the discovery context and confirmed product decisions so a
new task opened directly in the `leo-cli` workspace can continue without
re-reading the full MindCraft repository or repeating the product discussion.

The next task should read this note and the primary design, ask the user to
confirm any requested spec changes, and then create an implementation plan.

## User Workflow

- The command runs on a remote development or test server.
- The user enters that server with a normal SSH shell.
- The server is allowed to expose an internal HTTP port.
- Automatic local-browser launching is not required.
- `leo log` may print a clickable URL; the user will open it manually.
- The tool is intended for frequent searches over the most recent hour and
  occasional searches over the most recent few days.
- Total available logs on a server can exceed 1 GB.

## Why the Feature Is Worth Testing

The user currently experiences all of these problems:

- repeatedly changing include and exclude conditions in terminal searches;
- following a request by `search_id` or `user_id`;
- pausing, reviewing, and resuming a fast live stream;
- switching among many nested log files;
- reading long messages, payloads, and exception details.

The selected product boundary is a temporary browser-based reader, not a new
logging platform. The user explicitly chose on-demand scanning with a recent
one-hour default and no persistent index.

## MindCraft Evidence

The actual directory is `runtime/logs`, not `runtime/log`.

Observed files include:

```text
runtime/logs/api/log/error.log
runtime/logs/api/notice/notice.log
runtime/logs/api/prompt/prompt.log
runtime/logs/api/sensorsdata/access.log.YYYY-MM-DD
runtime/logs/api/sign/sign.log
runtime/logs/api/strategy/strategy.log
runtime/logs/api/view/view.tsv
runtime/logs/mc_ossclient.log
runtime/logs/mc_tosclient.log
```

The primary logger is configured in
`app/common/core/logger.py`. Its main Loguru format contains:

```text
timestamp | level | search_id | user_id | source - message
```

The inspected development config uses:

```text
LOG_LEVEL=INFO
LOG_ROTATION=5 MB
LOG_RETENTION=10
```

The local sample contained 204 physical lines, 13 distinct search IDs, and one
search ID associated with 143 lines. The longest sampled line was about 2.2 KB.
This supports click-to-filter request correlation and collapsible long messages.

The logs contain sensitive values such as user identifiers, tokens, phone
numbers, payment URLs, and payment-session data. No raw sensitive values are
copied into these documents. This is why configured directory boundaries,
temporary authentication, and no third-party collection are required.

## Confirmed CLI Integration

`leo-cli` is a Go 1.25 Cobra application. It already has:

- YAML configuration at `~/.config/leo-cli/config.yaml`;
- path expansion in `internal/config`;
- a single-binary release workflow;
- embedded pure-Go SQLite for repository metadata.

The log feature must be part of this repository and exposed as:

```bash
leo log
```

SQLite was discussed but explicitly deferred for version one. Do not persist
log records, indexes, UI preferences, saved queries, file catalogs, offsets,
tokens, or sessions in the initial implementation.

## Confirmed Configuration

Keys were intentionally shortened for convenient manual editing:

```yaml
proj:
  mindcraft:
    logs:
      - runtime/logs
      - /docker-runtime
```

The project key is the default path match. An optional override supports a
short alias:

```yaml
proj:
  mc:
    match: mindcraft
    logs:
      - runtime/logs
```

Project identification starts at `PWD` and walks upward. The nearest ancestor
directory whose name contains the match string is the project root. Relative
log paths resolve from that root, not from the current child directory.

## Confirmed File Discovery

- Recursively scan only configured log roots.
- Include only regular text files with common log suffixes.
- Initial suffixes: `.log`, rotated `.log.*`, `.out`, `.err`, `.tsv`, `.jsonl`,
  and `.ndjson`.
- Skip hidden files, symlinks, compressed files, and binary content.
- Use opaque file IDs in browser APIs and revalidate the allowed-root boundary
  on every read.

## Confirmed Search and Follow Behavior

- Default to the most recent hour.
- Initially select all eligible log files modified within that time range, then
  let the user narrow the scope from the file tree.
- Scan files on demand without a persistent index.
- Stream search results and progress before the complete scan finishes.
- Support literal matching, explicit regex mode, include terms, exclude terms,
  and structured filters for parsed fields.
- Allow cancellation and enforce bounded workers, duration, results, line size,
  and client buffers.
- Follow appends with SSE.
- Handle truncate, replacement, deletion, permission changes, and rename-based
  rotation with visible system records.
- Scrolling away from the bottom pauses automatic scrolling but not receipt of
  a bounded number of new records.

## Approved UI Direction

The user approved a dense, single-screen operational workspace:

- file tree and level shortcuts on the left;
- time range, search, field filters, progress, and cancellation at the top;
- structured log rows in the center;
- follow state, buffered record count, and jump-to-latest control at the
  bottom.

The user also approved the architecture boundary where log files remain the
only source of truth and the temporary service owns no database or index.

## Technical Direction

- Keep the feature inside the existing Go binary.
- Add `cmd/log.go`, `internal/project`, `internal/logview`, and `internal/logweb`.
- Embed static HTML, CSS, and JavaScript with `go:embed`.
- Do not add Node or npm to the build.
- Implement scanning in Go; do not require `rg` on the server.
- Use streamed fetch/NDJSON for structured historical search and SSE for live
  follow.
- Default to loopback; require an explicit flag for an internal-network bind.
- Exchange a one-time bootstrap URL token for an in-memory session cookie and
  redirect to a clean URL.

## Explicitly Deferred

- SQLite state or saved searches.
- Full-text indexes.
- Cross-server aggregation.
- Alerts and dashboards.
- Compressed-log search.
- Public or production internet exposure.
- A general-purpose query language.

## Next Step

The design discussion is complete. After the user reviews the two spec files,
invoke the `superpowers:writing-plans` skill and create a detailed implementation
plan in `docs/superpowers/plans/`. Implementation should then follow the plan
with test-driven development and verification before completion.
