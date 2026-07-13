# Hide Unparsed Logs by Default

**Date:** 2026-07-13

## Design

Keep the existing `Show unparsed` filter, but leave its checkbox unchecked on
initial page load. Historical searches will therefore send
`includeUnparsed: false`, and Follow will ignore records with `parsed: false`.
Users can still check the control to restore the current behavior.

Only the checkbox default and its existing HTML assertion need to change. The
search API, parser, Follow stream, and filtering logic remain unchanged.

## Verification

Update the workspace HTML test first and confirm it fails while the checkbox is
still checked. Remove the default `checked` attribute, then run the focused
`internal/logweb` tests and the full Go test suite.
