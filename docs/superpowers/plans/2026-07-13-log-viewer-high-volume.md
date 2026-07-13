# Log Viewer High-Volume Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Search large logs backward with bounded CPU, results, network flushes, and browser rows while defaulting to the last 5 minutes.

**Architecture:** Replace the forward file loop with a standard-library `ReadAt` reverse-line iterator and process records in timestamp groups so continuation-line inheritance is preserved. Keep the existing streamed search and browser UI, but align their limits and remove per-row layout reads.

**Tech Stack:** Go 1.25, `os.File.ReadAt`, `net/http`, embedded vanilla JavaScript

---

### Task 1: Reverse historical file scanning

**Files:**
- Modify: `internal/logview/search.go`
- Test: `internal/logview/search_test.go`

- [ ] **Step 1: Add failing reverse-search tests**

Add tests that create timestamped files and assert:

```go
func TestSearchReturnsRecordsNewestFirst(t *testing.T) {
	contents := "2026-07-13 10:00:00 | INFO | old | user | api - old\n" +
		"2026-07-13 10:04:00 | INFO | middle | user | api - middle\n" +
		"2026-07-13 10:05:00 | INFO | newest | user | api - newest\n"
	events, err := collectSearch(t, NewSearcher(testCatalogWithLine(t, contents)), Query{})
	if err != nil { t.Fatal(err) }
	records := searchRecords(events)
	if got := []string{records[0].SearchID, records[1].SearchID, records[2].SearchID}; !slices.Equal(got, []string{"newest", "middle", "old"}) {
		t.Fatalf("search IDs = %v", got)
	}
}
```

Also cover exact offsets, an 8-byte block boundary, CRLF, a line longer than
`MaxLineBytes`, early stop before `Query.Start`, and continuation lines emitted
after their timestamped anchor with the inherited timestamp.

- [ ] **Step 2: Verify the new tests fail**

Run: `go test ./internal/logview -run 'TestSearchReturnsRecordsNewestFirst|TestReverseLines|TestSearchReverse' -count=1`

Expected: FAIL because search is forward-only and no reverse-line helper exists.

- [ ] **Step 3: Implement the reverse-line iterator**

In `search.go`, add an internal record and helper:

```go
type reverseLine struct {
	offset    int64
	data      []byte
	truncated bool
}

func scanReverseLines(ctx context.Context, file *os.File, size int64, maxLineBytes, blockBytes int, emit func(reverseLine) error) (int64, error)
```

The helper uses one `blockBytes` read buffer, walks each block with
`bytes.LastIndexByte`, prepends cross-block fragments with a bounded copy, skips
only the synthetic empty line after a trailing newline, trims one trailing CR,
checks context between lines, and returns actual bytes read.

- [ ] **Step 4: Process timestamp groups newest-first**

Replace the forward `bufio.Reader` loop in `scanSearchFile` with
`scanReverseLines`. Accumulate reverse records until a record has a valid
timestamp, reverse that group to forward order, apply the existing inheritance
rule, then call `matcher.matches` and `emit`. Return a private sentinel when the
group timestamp is before `Query.Start`; treat that sentinel as successful early
completion for the file.

- [ ] **Step 5: Verify reverse-search tests pass**

Run: `go test ./internal/logview -run 'TestSearch|TestReverseLines' -count=1`

Expected: PASS.

### Task 2: Bound search CPU and results

**Files:**
- Modify: `internal/logview/search.go`
- Test: `internal/logview/search_test.go`

- [ ] **Step 1: Add failing default-limit and dense-allocation tests**

Assert `NewSearcher` uses two workers and 500 results. Add a dense short-line
fixture with 10,000 lines and compare `runtime.MemStats.TotalAlloc` before and
after a no-match search; require less than 64 MiB allocated.

- [ ] **Step 2: Verify the tests fail**

Run: `go test ./internal/logview -run 'TestNewSearcherUsesConservativeDefaults|TestSearchDenseShortLinesUsesBoundedAllocation' -count=1`

Expected: FAIL with current four-worker/10,000-result defaults or excessive allocation.

- [ ] **Step 3: Apply conservative defaults and candidate ordering**

Set `Workers: min(runtime.GOMAXPROCS(0), 2)` and `MaxResults: 500`. Sort resolved
candidate files by `ModTime` descending before dispatching jobs. Do not add
configuration keys.

- [ ] **Step 4: Verify the logview package**

Run: `go test ./internal/logview -count=1`

Expected: PASS.

### Task 3: Batch search response flushing

**Files:**
- Modify: `internal/logweb/server.go`
- Test: `internal/logweb/server_test.go`

- [ ] **Step 1: Add a failing HTTP flush-count test**

Wrap `httptest.ResponseRecorder` with a `Flush()` counter, search a fixture with
120 matching lines, and assert results are encoded while flush count is far below
the event count and includes the final done flush.

- [ ] **Step 2: Verify the test fails**

Run: `go test ./internal/logweb -run TestSearchBatchesResultFlushes -count=1`

Expected: FAIL because every event currently flushes.

- [ ] **Step 3: Flush result events in batches of 50**

In `handleSearch`, increment a pending-result counter after encoding a result.
Flush and reset at 50. Flush immediately for progress, warning, and done events.
Keep encoder errors and request cancellation unchanged.

- [ ] **Step 4: Verify the logweb package**

Run: `go test ./internal/logweb -count=1`

Expected: PASS.

### Task 4: Bound and lighten browser rendering

**Files:**
- Modify: `internal/logweb/assets/index.html`
- Modify: `internal/logweb/assets/app.js`
- Test: `internal/logweb/server_test.go`

- [ ] **Step 1: Add failing embedded-asset assertions**

Require `const MAX_ROWS = 500`, `rangeSeconds: 300`, the selected 300-second
option, a historical return before `getBoundingClientRect()`, and
`scheduleMessageDisclosureUpdate()` in the append path.

- [ ] **Step 2: Verify the assertions fail**

Run: `go test ./internal/logweb -run 'TestWorkspaceContainsOperationalControls|TestWorkspaceScript' -count=1`

Expected: FAIL with 1,500 rows, a one-hour default, and synchronous per-row layout reads.

- [ ] **Step 3: Apply the browser changes**

Set the row limit to 500 and initial range to 300 seconds. Select `Last 5
minutes` in HTML. In `insertNewestRow`, enforce the row limit and return for
historical rows before measuring height. Replace direct
`updateMessageDisclosure(messageRow)` with the existing scheduled updater.

- [ ] **Step 4: Verify focused browser behavior**

Run: `go test ./internal/logweb -count=1`

Expected: PASS.

### Task 5: Complete verification

**Files:**
- Modify: `.planning/2026-07-13-log-viewer-high-volume/task_plan.md`
- Modify: `.planning/2026-07-13-log-viewer-high-volume/progress.md`

- [ ] **Step 1: Format and run focused race tests**

Run: `gofmt -w internal/logview/search.go internal/logview/search_test.go internal/logweb/server.go internal/logweb/server_test.go`

Run: `go test -race ./internal/logview ./internal/logweb`

Expected: PASS.

- [ ] **Step 2: Run the full suite and static checks**

Run: `go test ./...`

Run: `go vet ./...`

Run: `git diff --check`

Expected: all commands pass.

- [ ] **Step 3: Run the large-file and browser smoke checks**

Run the integration-tagged 1 GiB search test, then serve a synthetic log with
more than 500 recent records and verify in the in-app browser that the table
stops at 500, the page remains interactive, and the default range reads `Last 5
minutes`.
