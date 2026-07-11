# Log Viewer Clear and Newest-First Design

**Date:** 2026-07-11

## Summary

Improve the embedded log workspace for monitoring newly arriving records. Add
short time ranges, make Clear establish a new search-time anchor without
discarding filters, and display newest records at the top while preserving the
reader's position when they inspect older records.

## Time Ranges

The Range control adds these options before the existing one-hour option:

- Last 1 minute
- Last 5 minutes
- Last 10 minutes

The existing one-hour and multi-hour/day options remain available. Selecting a
normal range computes `start = search time - range` and `end = search time`.

## Clear Behavior

The query action row gains a Clear button. Clicking it:

1. records the current browser time as `clearedAt`;
2. clears displayed records, waiting/discarded counters, and deduplication
   state;
3. preserves selected files, include/exclude terms, structured filters, regex,
   case sensitivity, and level filters;
4. leaves an active follow connection running;
5. changes the Range display to `Since clear HH:mm:ss`;
6. sets the next historical search range to `clearedAt -> search click time`.

Selecting any normal Range option removes the Clear anchor. Clear does not
automatically execute a historical search because its primary purpose is to
empty the current console and observe future follow records.

## Newest-First Records

All new records enter at the top of the table.

- Live follow records are ordered by receipt, with the newest receipt first.
- Historical results remain incremental while scanning. Each arriving result
  is inserted at the top.
- When historical search completes, records with parsed timestamps are ordered
  newest-first. Records without parsed timestamps remain newest-arrival-first.
- System and warning rows remain newest-arrival-first and are not included in
  timestamp sorting.

The browser continues to keep at most 1,500 table rows. With newest-first
ordering, overflow removes the oldest row from the bottom.

## Scroll Behavior

The top is the live edge.

- When the viewport is at the top, incoming live records remain visible.
- When the user scrolls down, incoming rows do not move the records currently
  being read. The UI compensates for the inserted row height and increments the
  waiting-record count.
- Jump to latest scrolls to the top, resets the waiting count, and resumes live
  positioning.

## Implementation Boundary

This is a browser-asset change in `internal/logweb/assets`. The Go search and
follow APIs already accept exact start/end timestamps and need no contract
change. Embedded-asset tests will assert the new controls exist. Playwright
verification will cover Clear state, short ranges, newest-first insertion,
scroll preservation, and the updated jump direction on desktop and mobile.
