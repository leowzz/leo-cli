# Log Viewer Clear and Newest-First Design

**Date:** 2026-07-11

## Summary

Improve the embedded log workspace for monitoring newly arriving records. Add
short time ranges, make Clear establish a new search-time anchor without
discarding filters, and display newest records at the top while preserving the
reader's position when they inspect older records.

## Time Ranges

The Range control becomes a split control:

- The left side is a button showing the active preset, such as
  `Last 1 minute`.
- The right side opens the preset menu.

The menu adds these options before the existing one-hour option:

- Last 1 minute
- Last 5 minutes
- Last 10 minutes

The existing one-hour and multi-hour/day options remain available. Selecting a
menu item updates the left button, removes any Clear anchor, and immediately
searches with `start = search time - range` and `end = search time`.

Clicking the left button reapplies its displayed preset, removes any Clear
anchor, and immediately searches. This provides a one-click way to leave Clear
mode without opening the menu.

## Clear Behavior

The query action row gains a Clear button. Clicking it:

1. records the current browser time as `clearedAt`;
2. clears displayed records, waiting/discarded counters, and deduplication
   state;
3. preserves selected files, include/exclude terms, structured filters, regex,
   case sensitivity, and level filters;
4. leaves an active follow connection running;
5. leaves the range button label unchanged;
6. displays `Cleared at HH:mm:ss` in the result status area;
7. sets the next historical search range to `clearedAt -> search click time`.

Selecting a Range menu item or clicking the current Range button removes the
Clear anchor and immediately searches. Clear itself does not execute a
historical search because its primary purpose is to empty the current console
and observe future follow records.

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

## Automatic Follow

After the file catalog loads, the browser starts the default historical search
and Follow concurrently. Historical search failure does not prevent Follow
from staying active.

Follow remains active across Clear, searches, and range changes. EventSource
continues to handle temporary network reconnection. When the user explicitly
clicks Stop follow, Follow stays disabled for the rest of that page session and
no later Clear or search action restarts it. Reloading or reopening the page
starts Follow automatically again.

## Implementation Boundary

This is a browser-asset change in `internal/logweb/assets`. The Go search and
follow APIs already accept exact start/end timestamps and need no contract
change. Embedded-asset tests will assert the new split range and Clear controls
exist. Playwright verification will cover Clear state, immediate preset
searches, one-click Clear exit, short ranges, newest-first insertion, scroll
preservation, automatic Follow with manual-stop persistence, and the updated
jump direction on desktop and mobile.
