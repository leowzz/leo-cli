# Hide Unparsed Logs by Default Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hide unparsed log records by default while preserving the existing opt-in checkbox.

**Architecture:** Change only the checkbox's initial HTML state. Existing JavaScript already reads that state for historical searches and Follow filtering.

**Tech Stack:** Go tests, embedded HTML, vanilla JavaScript

---

### Task 1: Change the default filter

**Files:**
- Modify: `internal/logweb/server_test.go`
- Modify: `internal/logweb/assets/index.html`

- [ ] **Step 1: Write the failing test**

Replace the workspace HTML assertion for `type="checkbox" checked` with an
assertion that the `show-unparsed` input does not contain `checked`.

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `go test ./internal/logweb -run TestWorkspaceContainsOperationalControls -count=1`

Expected: FAIL because the input still contains the `checked` attribute.

- [ ] **Step 3: Write the minimal implementation**

Change:

```html
<input id="show-unparsed" type="checkbox" checked>
```

to:

```html
<input id="show-unparsed" type="checkbox">
```

- [ ] **Step 4: Verify the focused and full suites**

Run: `go test ./internal/logweb -count=1`

Expected: PASS.

Run: `go test ./...`

Expected: PASS.
