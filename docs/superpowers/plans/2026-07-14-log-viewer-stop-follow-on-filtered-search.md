# Stop Follow On Filtered Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop live Follow before a user-triggered historical search when an Include, Exclude, Search ID, User ID, Source, or Level filter is active.

**Architecture:** Keep the existing `runSearch()` behavior unchanged for startup, time-range changes, and Show unparsed changes. Add one browser-side wrapper that derives active filters from the existing `searchQuery()`, stops Follow when any filter has values, and then runs the historical search.

**Tech Stack:** Go 1.24, embedded vanilla JavaScript, Go `testing`

## Global Constraints

- Only the Search button and Enter in the Include, Exclude, Search ID, User ID, or Source fields use the filtered-search action.
- The initial automatic search, time-range changes, and Show unparsed changes must not stop Follow.
- Regex and Case sensitive alone do not count as filters.
- A user may manually restart Follow after a filtered search.
- Do not change Go APIs, search queries, follow streams, persisted state, or dependencies.

---

### Task 1: Stop Follow For Manual Filtered Searches

**Files:**
- Modify: `internal/logweb/assets/app.js:307-326,771-799`
- Test: `internal/logweb/server_test.go:327`

**Interfaces:**
- Consumes: `searchQuery() Object`, `stopFollow() void`, and `runSearch() Promise<void>` from `internal/logweb/assets/app.js`.
- Produces: `runFilteredSearch() void`, used only by the Search button click and search-field Enter handlers.

- [ ] **Step 1: Write the failing embedded-asset test**

Add this test before `TestWorkspaceUsesBoundedFiveMinuteDefaults` in `internal/logweb/server_test.go`:

```go
func TestWorkspaceStopsFollowForManualFilteredSearch(t *testing.T) {
	script, err := embeddedAssets.ReadFile("assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`const filters = [query.include, query.exclude, query.searchIds, query.userIds, query.sources, query.levels];`,
		`elements.search.addEventListener("click", runFilteredSearch);`,
		`if (event.key === "Enter") runFilteredSearch();`,
		`elements.rangeApply.addEventListener("click", applySelectedRange);`,
		`elements.showUnparsed.addEventListener("change", runSearch);`,
	} {
		if !bytes.Contains(script, []byte(required)) {
			t.Errorf("workspace script is missing %q", required)
		}
	}

	wrapperAt := bytes.Index(script, []byte("function runFilteredSearch() {"))
	if wrapperAt < 0 {
		t.Fatal("workspace script is missing runFilteredSearch")
	}
	wrapper := script[wrapperAt:]
	endAt := bytes.Index(wrapper, []byte("\nasync function runSearch()"))
	if endAt < 0 {
		t.Fatal("runFilteredSearch is not immediately before runSearch")
	}
	wrapper = wrapper[:endAt]
	stopAt := bytes.Index(wrapper, []byte("stopFollow();"))
	searchAt := bytes.Index(wrapper, []byte("runSearch();"))
	if stopAt < 0 || searchAt < 0 || stopAt > searchAt {
		t.Fatal("filtered search does not stop Follow before searching")
	}
}
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
rtk test go test ./internal/logweb -run TestWorkspaceStopsFollowForManualFilteredSearch -count=1
```

Expected: FAIL because `runFilteredSearch` and its event-handler wiring do not exist.

- [ ] **Step 3: Add the minimal filtered-search wrapper and wire manual actions**

Add this function immediately after `searchQuery()` in `internal/logweb/assets/app.js`:

```javascript
function runFilteredSearch() {
  const query = searchQuery();
  const filters = [query.include, query.exclude, query.searchIds, query.userIds, query.sources, query.levels];
  if (filters.some((values) => values.length)) stopFollow();
  runSearch();
}
```

Change only the Search click and search-field Enter handlers:

```javascript
elements.search.addEventListener("click", runFilteredSearch);
```

```javascript
for (const input of [elements.include, elements.exclude, elements.searchID, elements.userID, elements.source]) {
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") runFilteredSearch();
  });
}
```

Leave these existing handlers unchanged:

```javascript
elements.rangeApply.addEventListener("click", applySelectedRange);
elements.rangeMenu.addEventListener("change", applySelectedRange);
elements.showUnparsed.addEventListener("change", runSearch);
```

- [ ] **Step 4: Run focused tests and verify GREEN**

Run:

```bash
rtk test go test ./internal/logweb -run 'TestWorkspace(StopsFollowForManualFilteredSearch|ScriptStartsFollowAfterCatalog)' -count=1
```

Expected: PASS. The manual filtered-search behavior is present and automatic Follow still starts after the initial search.

- [ ] **Step 5: Run repository verification**

Run:

```bash
rtk test go test ./...
rtk git diff --check
```

Expected: all Go tests pass and `git diff --check` prints no errors.

- [ ] **Step 6: Commit the implementation**

```bash
rtk git add internal/logweb/assets/app.js internal/logweb/server_test.go
rtk git commit -m "fix: stop follow for filtered log searches"
```
