# Log Viewer Unparsed Records Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let operators include or exclude unparsed log records and assign unparsed continuation lines the nearest earlier valid timestamp from the same file.

**Architecture:** Historical scans and Follow each maintain an independent last-known timestamp per file and apply it before filtering or emission. Historical search filters `parsed: false` through a new query field, while the browser filters Follow events locally so toggling never reconnects or changes Follow state.

**Tech Stack:** Go, embedded HTML/CSS/JavaScript, Go testing, Playwright CLI

---

## File Structure

- `internal/logview/search.go`: add the query field, filter unparsed records, and inherit timestamps during each file scan.
- `internal/logview/search_test.go`: cover search filtering, inherited ranges, timeless leading records, and file isolation.
- `internal/logview/follow.go`: retain the last valid timestamp per `followState`, apply it before emission, and reset it on replacement/truncation.
- `internal/logview/follow_test.go`: cover Follow inheritance and reset behavior.
- `internal/logweb/assets/index.html`: add the default-checked `Show unparsed` control.
- `internal/logweb/assets/app.js`: send the search field, refresh on change, and filter incoming Follow records.
- `internal/logweb/server_test.go`: assert the embedded UI and script contract.

### Task 1: Historical Timestamp Inheritance and Filtering

**Files:**
- Modify: `internal/logview/search.go`
- Test: `internal/logview/search_test.go`

- [ ] **Step 1: Write failing historical-search tests**

Add focused tests that create a valid structured line followed by an unparsed exception line. Assert that the exception remains `Parsed == false`, inherits the valid line's `Timestamp`, and is included or excluded by the inherited timestamp range. Add cases asserting that a leading unparsed line stays timeless, another file cannot provide its timestamp, and `Query{IncludeUnparsed: false}` excludes all unparsed records.

```go
func TestSearchInheritsTimestampForUnparsedRecords(t *testing.T) {
    // Write a parsed 10:00 record followed by "stack trace".
    // Search 09:59-10:01 with IncludeUnparsed true.
    // Assert the second record is unparsed and has the 10:00 timestamp.
}

func TestSearchCanExcludeUnparsedRecords(t *testing.T) {
    // Search the same file with IncludeUnparsed false.
    // Assert only the parsed record is returned.
}
```

- [ ] **Step 2: Run tests and verify the expected failures**

```bash
go test ./internal/logview -run 'TestSearch(Inherits|KeepsLeading|Isolates|CanExclude)' -count=1
```

Expected: FAIL because `Query.IncludeUnparsed` and timestamp inheritance do not exist.

- [ ] **Step 3: Add the query field and minimal matcher rule**

Add the JSON field to `Query` and reject unparsed records before other matcher checks when it is false:

```go
IncludeUnparsed bool `json:"includeUnparsed"`

if !m.query.IncludeUnparsed && !record.Parsed {
    return false
}
```

Update existing tests that intentionally search raw lines to set `IncludeUnparsed: true`; this makes the new API default explicit without changing structured-record tests.

- [ ] **Step 4: Apply per-file timestamp inheritance before matching**

Inside `scanSearchFile`, keep `var lastTimestamp *time.Time`. After `ParseLine`, update it from records with a non-nil timestamp; otherwise copy it onto unparsed records only:

```go
if record.Timestamp != nil {
    inherited := *record.Timestamp
    lastTimestamp = &inherited
} else if !record.Parsed && lastTimestamp != nil {
    inherited := *lastTimestamp
    record.Timestamp = &inherited
}
```

Because the variable is local to one `scanSearchFile` invocation, concurrent files remain isolated.

- [ ] **Step 5: Run the focused and package tests**

```bash
go test ./internal/logview -run 'TestSearch(Inherits|KeepsLeading|Isolates|CanExclude)' -count=1
go test ./internal/logview -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the historical-search change**

```bash
git add internal/logview/search.go internal/logview/search_test.go
git commit -m "feat: filter and timestamp unparsed search records"
```

### Task 2: Follow Timestamp Inheritance and Reset

**Files:**
- Modify: `internal/logview/follow.go`
- Test: `internal/logview/follow_test.go`

- [ ] **Step 1: Write failing Follow tests**

Add a unit test for `consumeFollowBytes` that feeds a valid timestamped line and an unparsed continuation, then asserts the second record inherits the timestamp while remaining unparsed. Extend truncation and rotation coverage so an unparsed first line after reset has a nil timestamp.

```go
func TestConsumeFollowBytesInheritsTimestampForUnparsedRecords(t *testing.T) {
    state := &followState{file: File{ID: "file-1", RelativePath: "app.log"}}
    // Consume a timestamped structured line and "stack trace\n".
    // Assert equal timestamps and Parsed false on the continuation.
}
```

- [ ] **Step 2: Run tests and verify they fail**

```bash
go test ./internal/logview -run 'Test(Follow.*Reset|ConsumeFollowBytesInherits)' -count=1
```

Expected: FAIL because `followState` does not retain or clear a timestamp.

- [ ] **Step 3: Add timestamp state and a shared emission helper**

Add `lastTimestamp *time.Time` to `followState`. Introduce a helper used by newline emission and `flushFollowPending`:

```go
func followRecord(state *followState, record Record) Record {
    if record.Timestamp != nil {
        timestamp := *record.Timestamp
        state.lastTimestamp = &timestamp
    } else if !record.Parsed && state.lastTimestamp != nil {
        timestamp := *state.lastTimestamp
        record.Timestamp = &timestamp
    }
    return record
}
```

Set `state.lastTimestamp = nil` in both rotation replacement and truncation reset blocks before reading the new generation.

- [ ] **Step 4: Run the focused and package tests**

```bash
go test ./internal/logview -run 'Test(Follow.*Reset|ConsumeFollowBytesInherits)' -count=1
go test ./internal/logview -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the Follow change**

```bash
git add internal/logview/follow.go internal/logview/follow_test.go
git commit -m "feat: inherit timestamps while following logs"
```

### Task 3: Show Unparsed Browser Control

**Files:**
- Modify: `internal/logweb/assets/index.html`
- Modify: `internal/logweb/assets/app.js`
- Test: `internal/logweb/server_test.go`

- [ ] **Step 1: Write failing embedded-asset tests**

Extend the workspace HTML test to require a checked checkbox with `id="show-unparsed"`. Extend the script test to require:

```js
includeUnparsed: elements.showUnparsed.checked
elements.showUnparsed.addEventListener("change", runSearch)
if (!record.parsed && !elements.showUnparsed.checked) return
```

- [ ] **Step 2: Run the focused tests and verify they fail**

```bash
go test ./internal/logweb -run 'TestWorkspace(ContainsOperationalControls|ScriptSupportsUnparsedFilter)' -count=1
```

Expected: FAIL because the control and script behavior are absent.

- [ ] **Step 3: Add the default-checked control**

Place this toggle with the existing level filters:

```html
<label class="toggle unparsed-toggle">
  <input id="show-unparsed" type="checkbox" checked>
  <span>Show unparsed</span>
</label>
```

Reuse the existing toggle styling unless browser verification shows narrow-viewport overflow.

- [ ] **Step 4: Wire search refresh and live filtering**

Add `showUnparsed` to `elements`, send its checked value as `includeUnparsed`, register `change` with `runSearch`, and return at the start of `appendRecord` when an unparsed record is hidden. Do not call `startFollow`, `stopFollow`, or recreate EventSource from this handler.

- [ ] **Step 5: Run focused tests and JavaScript syntax validation**

```bash
go test ./internal/logweb -run 'TestWorkspace(ContainsOperationalControls|ScriptSupportsUnparsedFilter)' -count=1
node --check internal/logweb/assets/app.js
```

Expected: PASS.

- [ ] **Step 6: Commit the browser control**

```bash
git add internal/logweb/assets/index.html internal/logweb/assets/app.js internal/logweb/server_test.go
git commit -m "feat: toggle unparsed log records"
```

### Task 4: Integrated Verification

**Files:**
- Modify only if verification identifies a defect in the files above.

- [ ] **Step 1: Run formatting and static checks**

```bash
gofmt -w internal/logview internal/logweb
go vet ./...
node --check internal/logweb/assets/app.js
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 2: Run full and race tests**

```bash
go test ./... -count=1
go test -race ./internal/logview ./internal/logweb ./cmd -count=1
```

Expected: all packages PASS with no race reports.

- [ ] **Step 3: Verify the browser workflow with Playwright CLI**

Start an isolated demo server with a valid record followed by an exception line. Confirm the page initially displays both rows, the exception row shows the inherited timestamp, disabling `Show unparsed` immediately removes it, and enabling it restores it. Repeat the toggle once with Follow active and once after Stop follow; assert the Follow status never changes because of the toggle. Confirm zero browser console errors and warnings on desktop and a 390px mobile viewport.

- [ ] **Step 4: Scan for sensitive names and generated artifacts**

```bash
if rg -n -i 'mind''craft' . -g '!bin/**' -g '!.git/**'; then exit 1; fi
git status --short
```

Expected: no sensitive-name matches and only intended changes. Remove `.playwright-cli`, `output`, and temporary demo directories.

- [ ] **Step 5: Commit any verification fixes**

If verification required source changes, repeat the relevant focused and full checks, then commit only those fixes. If no changes were required, do not create an empty commit.
