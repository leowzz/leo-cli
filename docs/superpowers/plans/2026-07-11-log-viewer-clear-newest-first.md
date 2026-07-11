# Log Viewer Clear and Newest-First Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Clear time anchor, short range presets, a split range control, and newest-first log flow to the embedded log workspace.

**Architecture:** Keep the Go HTTP/search contracts unchanged. Store the selected preset and optional Clear timestamp in browser state, derive exact query start/end values at search time, and change DOM insertion/retention/scroll logic so the top is the live edge.

**Tech Stack:** Embedded HTML/CSS/vanilla JavaScript, Go `httptest`, Playwright CLI.

---

## File Structure

- `internal/logweb/assets/index.html`: add the split range control, short presets, and Clear command.
- `internal/logweb/assets/app.css`: size and style the split control without changing the dense toolbar layout.
- `internal/logweb/assets/app.js`: manage preset/Clear state, immediate searches, newest-first insertion, completed-search sorting, and top-edge scroll behavior.
- `internal/logweb/server_test.go`: assert the embedded workspace contains the new controls and preset values.
- `README.md`, `README.zh.md`: document short ranges, Clear, and newest-first follow behavior.

## Task 1: Split Range and Clear Anchor

- [ ] Extend `TestWorkspaceContainsOperationalControls` to require `range-apply`, `range-menu`, `clear`, and values `60`, `300`, and `600`.
- [ ] Run `go test ./internal/logweb -run TestWorkspaceContainsOperationalControls -count=1` and confirm it fails on missing controls.
- [ ] Replace the single select with a left apply button and right preset select; add the Clear button.
- [ ] Add `state.rangeSeconds` and `state.clearedAt`. Make `searchQuery()` use `clearedAt` when set, otherwise subtract `rangeSeconds`.
- [ ] Make preset selection and the apply button clear `clearedAt` and call `runSearch()` immediately.
- [ ] Make Clear abort any active historical search, preserve filters/follow, clear records, set `clearedAt`, and display `Cleared at HH:mm:ss`.
- [ ] Run the focused Go test and confirm it passes.

## Task 2: Newest-First Table and Top Live Edge

- [ ] Change record insertion from append to prepend and overflow removal from first row to last row.
- [ ] Track arrival order on record rows and, when search completes, sort timestamped rows descending while keeping untimestamped rows newest-arrival-first.
- [ ] Replace bottom-distance detection with `scrollTop < 32`; make Jump to latest set `scrollTop = 0`.
- [ ] When the user is below the top, preserve the visible viewport by adding the inserted row-height delta to `scrollTop` and increment the waiting count.
- [ ] Update labels/status text so the top is consistently treated as the live edge.
- [ ] Run `go test ./internal/logweb -count=1`.

## Task 3: Documentation and Browser Verification

- [ ] Document the 1/5/10-minute presets, Clear anchor, split-control behavior, and newest-first table in both READMEs.
- [ ] Start a temporary configured demo server and authenticate with its bootstrap URL.
- [ ] Use Playwright to verify selecting a preset immediately searches, Clear empties the table without changing filters or the range button, and clicking the range button exits Clear mode.
- [ ] Append live records and verify newest appears first, scrolling down preserves the current viewport, the waiting count increments, and Jump to latest returns to the top.
- [ ] Capture and inspect desktop and 390px mobile screenshots for overlap and overflow, then remove generated artifacts.
- [ ] Run `gofmt -w internal/logweb`, `go vet ./...`, `go test ./... -count=1`, and `go test -race ./internal/logweb ./internal/logview ./cmd -count=1`.
- [ ] Review `git diff --check` and verify no sensitive example names or generated files are present.

## Task 4: Automatic Follow

- [x] Extend the embedded-script test to require the catalog success path to call `runSearch()` and then `startFollow()`.
- [x] Run `go test ./internal/logweb -run TestWorkspaceScriptStartsFollowAfterCatalog -count=1` and confirm it fails before the startup call exists.
- [x] Start Follow immediately after launching the initial historical search in `loadCatalog()`.
- [x] Keep `startFollow()` out of Clear, manual Search, and range handlers so Stop follow persists for the page session.
- [x] Use Playwright to verify the page opens in `Live` state, Stop follow remains stopped after Clear and range search, and a reload starts Follow again.
- [x] Run the focused test, full Go tests, race tests, JS syntax check, and browser console check.
