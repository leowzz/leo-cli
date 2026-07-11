# Log Viewer Unparsed Records Design

**Date:** 2026-07-11

## Summary

Add an explicit `Show unparsed` filter to the log viewer and give unparsed
records a useful time when their file contains an earlier parsed record. This
keeps exception stacks and other continuation lines visible by default while
allowing operators to hide them when they need a strictly structured view.

## User Interface

The Levels area gains a `Show unparsed` checkbox. It is selected by default so
the viewer continues to show every record on first load.

Changing the checkbox immediately runs the current historical search. It does
not start, stop, or reconnect Follow. A manually stopped Follow therefore
remains stopped, while an active Follow remains connected.

When the option is disabled:

- the refreshed historical result excludes records with `parsed: false`;
- future unparsed Follow records are ignored by the browser; and
- parsed records and all existing filters retain their current behavior.

When the option is enabled, the refreshed historical result and future Follow
events include unparsed records again.

## Timestamp Inheritance

Each log file has an independent last-known timestamp while it is read. A
record with a successfully parsed timestamp updates that value. When a later
line cannot be parsed as a structured record, it inherits the last-known
timestamp while remaining marked `parsed: false`.

The inheritance rules are:

- inheritance only moves forward from an earlier line in the same file;
- files never share a timestamp;
- a structured record without a valid timestamp does not update the anchor;
- an unparsed line before the first valid timestamp remains timeless; and
- Follow clears the file's timestamp anchor after truncation or rotation.

The inherited value uses the existing `timestamp` JSON field. No additional
display-only timestamp field is needed. The `parsed` field remains the source
of truth for filtering and ensures inherited records are not mistaken for
successfully parsed records.

## Historical Search

The search query gains `includeUnparsed`, sent as `true` by the default UI.
The search matcher excludes `parsed: false` records when the value is false.

Timestamp inheritance happens while each file is scanned, before query
matching. An inherited timestamp therefore participates in the existing start
and end checks. A timeless unparsed record keeps the current compatibility
behavior: it is included when `includeUnparsed` is true even if the query has a
time range, because there is no defensible timestamp with which to reject it.

The per-file scan owns its last-known timestamp, which naturally isolates
concurrent file scans from one another.

## Follow

The follow reader maintains a last-known timestamp for each open file. It
applies the same inheritance rule before emitting a record. Rotation or
truncation resets that file's anchor so records from a new file generation
cannot inherit stale time.

The Follow HTTP API remains unchanged and continues to stream all records.
The browser applies `Show unparsed` when handling each Follow event. This
avoids reconnecting EventSource when the checkbox changes and preserves the
existing manual Stop follow behavior.

## Error Handling

Timestamp inheritance is best-effort and never turns an unparsed record into a
parsed record. If no valid anchor exists, the record is emitted without a
timestamp. Existing line truncation, scan errors, cancellation, and Follow
reconnection behavior remain unchanged.

## Verification

Automated tests will cover:

- historical inheritance from the nearest earlier valid timestamp;
- timeless unparsed records at the start of a file;
- isolation between files;
- exclusion when `includeUnparsed` is false;
- Follow inheritance and reset after truncation or rotation;
- the default checked control and search request field;
- immediate search when the control changes; and
- browser-side filtering of incoming Follow records without changing Follow
  connection state.

Browser verification will confirm that toggling the option immediately
refreshes visible records, inherited timestamps render in the Time column, and
active or manually stopped Follow state is preserved.
