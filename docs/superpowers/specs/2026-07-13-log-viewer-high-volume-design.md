# Log Viewer High-Volume Design

**Date:** 2026-07-13

## Goal

Keep `leo log` responsive when recent log files are large or dense. Historical
search must avoid sustained multi-core CPU usage, stream only a bounded useful
result set, and keep the browser main thread responsive.

## Server-Side Reverse Search

Historical search reads each candidate file backward from EOF in fixed 64 KiB
blocks. The scanner reconstructs lines that cross block boundaries, preserves
their byte offsets, and keeps the existing 256 KiB per-line truncation limit.
It uses `ReadAt` and bounded reusable buffers; it does not load a whole file.

Reverse traversal encounters unparsed continuation lines before their preceding
timestamped record. The scanner therefore buffers one timestamp group, restores
that group to forward line order, and applies the existing timestamp inheritance
and query matcher before emitting it. `Show unparsed` behavior remains unchanged.

Within a file, timestamped logs are assumed to be chronological. After the
scanner reaches a valid timestamp earlier than the query start, it stops reading
that file. Records newer than the query end are skipped while scanning continues.
A file without valid timestamps cannot use the time cutoff and is scanned to its
start, subject to the shared timeout and result limit.

Candidate files are ordered by modification time descending. At most two files
are scanned concurrently. Results are newest-first within each file and remain
approximately newest-first across files, matching the existing concurrent result
semantics. The browser performs its existing final timestamp sort.

## Resource Limits

The server result limit and browser row limit are both 500. Reaching 500 results
cancels remaining scan work and reports the existing `limit` completion reason.

The search response flushes after 50 result events and immediately after
progress, warning, and done events. This preserves incremental delivery while
avoiding a network flush for every line.

The default search duration and line-size limits remain unchanged. No persistent
index, copied log data, fixed tail-byte window, or new dependency is introduced.

## Browser Work

The initial range is the last 5 minutes. Initial file selection uses the same
5-minute cutoff, and the range control displays `Last 5 minutes`.

Historical rows do not measure their height because that value is only needed
to preserve scroll position for live Follow rows. Message overflow measurement
is coalesced through the existing animation-frame scheduler instead of running
synchronously for every inserted historical row.

Follow retains its current protocol and behavior. Its visible row buffer also
uses the shared 500-row browser limit.

## Error Handling

Cancellation, timeout, unreadable-file warnings, invalid regex errors, line
truncation, rotation, and session handling keep their current behavior. Reverse
block reads propagate file errors through the existing per-file warning path.

The early time cutoff relies on chronological timestamps within a log file. If
that assumption later proves false for a supported log source, that source will
need a configurable full-scan mode; this version does not add speculative UI or
configuration for it.

## Verification

Automated tests cover:

- newest-first reverse scanning with exact offsets;
- lines crossing block boundaries and oversized-line truncation;
- start/end time filtering and early file cutoff;
- unparsed continuation timestamp inheritance;
- cancellation and the 500-result default;
- dense short-line scans without the previous 64 KiB-per-line allocation;
- two-worker, 5-minute, and 500-row defaults;
- batched search flushing; and
- historical DOM insertion without per-row height or overflow measurement.

The focused log packages, full Go suite, race tests, and a browser smoke test
with more than 500 synthetic records must pass before completion.
