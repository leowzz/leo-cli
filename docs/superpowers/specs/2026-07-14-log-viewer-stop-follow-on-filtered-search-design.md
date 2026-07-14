# Stop Follow On Filtered Search Design

## Goal

Stop live Follow immediately when the user starts a filtered historical search,
so new unfiltered records cannot mix into the filtered result set.

## Behavior

The Search button and Enter in the Include, Exclude, Search ID, User ID, or
Source fields run a filtered search action. Before searching, that action stops
Follow when at least one of those fields is non-empty or at least one Level is
selected.

Regex and Case sensitive do not count as filters by themselves because they do
not constrain a search without an Include term.

The initial automatic search, time-range changes, and Show unparsed changes
continue to use the normal search action and do not stop Follow. A user may
start Follow again manually after a filtered search.

## Implementation

Keep the change in `internal/logweb/assets/app.js`. Add one small wrapper around
the existing `runSearch()` function. The wrapper detects active filters, calls
the existing `stopFollow()` when necessary, and then calls `runSearch()`.

Wire only the Search button and search-field Enter handlers to the wrapper.
Leave startup, range, and Show unparsed handlers wired directly to
`runSearch()`.

No Go API, search query, follow stream, or persisted state changes are needed.

## Verification

Add a focused embedded-asset test that checks the wrapper stops Follow before
searching and that only the manual Search and Enter handlers use it. Run the
focused `internal/logweb` test and the full Go test suite.

