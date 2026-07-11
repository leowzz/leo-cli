# Log Viewer Table Interactions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add resizable log columns, selectable Message content, and explicit copy/filter menus for structured fields and Message.

**Architecture:** Header resize handles update the embedded table's `<col>` widths without rebuilding rows. Each data cell owns a zero-width overlay trigger, while one fixed-position menu attached to `body` serves every row and avoids table clipping. Message text is a selectable element and only its disclosure button changes collapsed state.

**Tech Stack:** Go embedded assets and tests, vanilla HTML/CSS/JavaScript, Pointer Events, Clipboard API with HTTP fallback, Playwright CLI

---

## File Structure

- `internal/logweb/assets/index.html`: identify the table/columns and add semantic resize handles plus the reusable action-menu host.
- `internal/logweb/assets/app.css`: style resize handles, selectable cells, disclosure states, hover-only overlay triggers, and the fixed menu.
- `internal/logweb/assets/app.js`: implement resizing, message disclosure measurement, structured-field actions, clipboard actions, menu placement/dismissal, and keyboard behavior.
- `internal/logweb/server_test.go`: assert the embedded HTML, CSS, and script contain the interaction contract.

### Task 1: Embedded Interaction Contract

**Files:**
- Modify: `internal/logweb/server_test.go`
- Test: `internal/logweb/server_test.go`

- [ ] **Step 1: Add a failing HTML/CSS contract test**

Add `TestWorkspaceContainsResizableTableAndActionMenu`. Fetch `/`, `/app.css`, and `/app.js` through an authenticated test server and assert these exact contract markers:

```go
htmlRequired := []string{
    `id="log-table"`,
    `data-column="time"`,
    `data-column="level"`,
    `data-column="search-id"`,
    `data-column="user-id"`,
    `data-column="source"`,
    `class="column-resize"`,
    `id="cell-action-menu"`,
    `role="menu"`,
}
cssRequired := []string{
    `.column-resize`,
    `cursor: col-resize`,
    `.cell-action-trigger`,
    `.message-text`,
    `.message-row.expanded`,
    `.cell-action-menu`,
}
scriptRequired := []string{
    `initColumnResizing()`,
    `openCellActionMenu`,
    `closeCellActionMenu`,
    `updateMessageDisclosure`,
    `copyText`,
}
```

Use the existing bootstrap-cookie pattern and report each missing marker with the asset name.

- [ ] **Step 2: Run the focused test and verify RED**

```bash
go test ./internal/logweb -run TestWorkspaceContainsResizableTableAndActionMenu -count=1
```

Expected: FAIL with missing table, resize, menu, Message, and clipboard markers.

- [ ] **Step 3: Commit only after later tasks make the contract pass**

Do not commit a failing test. Keep it uncommitted through Tasks 2 and 3 so each production change is driven by this observed failure.

### Task 2: Resizable Columns

**Files:**
- Modify: `internal/logweb/assets/index.html`
- Modify: `internal/logweb/assets/app.css`
- Modify: `internal/logweb/assets/app.js`
- Test: `internal/logweb/server_test.go`

- [ ] **Step 1: Mark table columns and add resize controls**

Give the table `id="log-table"`. Add stable `data-column` values to all six `<col>` elements. Add a resize button to the first five headers:

```html
<th scope="col">
  Time
  <button class="column-resize" type="button"
          data-column="time" aria-label="Resize Time column"></button>
</th>
```

Repeat with `level`, `search-id`, `user-id`, and `source`. Message has no right-edge handle because its width is the remaining track.

Add `logTable: document.querySelector("#log-table")` to `elements`.

- [ ] **Step 2: Define stable widths and resize affordances**

Keep the current default widths. Add minimums through each resizable `<col>` dataset:

```html
<col class="col-time" data-column="time" data-min-width="120">
<col class="col-level" data-column="level" data-min-width="60">
<col class="col-id" data-column="search-id" data-min-width="90">
<col class="col-id" data-column="user-id" data-min-width="90">
<col class="col-source" data-column="source" data-min-width="110">
<col class="col-message" data-column="message">
```

Style `.column-resize` as an absolute 7px-wide header-edge button with `cursor: col-resize`, a visible hover/focus/drag indicator, and `touch-action: none`. Keep header text selectable state unchanged and add right padding so the handle does not cover the label.

- [ ] **Step 3: Implement pointer resizing**

Add `initColumnResizing()` and call it once before `loadCatalog()`:

```js
function initColumnResizing() {
  for (const handle of elements.logTable.querySelectorAll(".column-resize")) {
    handle.addEventListener("pointerdown", (event) => {
      const column = elements.logTable.querySelector(
        `col[data-column="${handle.dataset.column}"]`,
      );
      const startX = event.clientX;
      const startWidth = column.getBoundingClientRect().width;
      const minWidth = Number(column.dataset.minWidth);
      handle.setPointerCapture(event.pointerId);
      handle.classList.add("dragging");

      const move = (moveEvent) => {
        column.style.width = `${Math.max(minWidth, startWidth + moveEvent.clientX - startX)}px`;
        updateMessageDisclosure();
      };
      const finish = () => {
        handle.classList.remove("dragging");
        handle.removeEventListener("pointermove", move);
        handle.removeEventListener("pointerup", finish);
        handle.removeEventListener("pointercancel", finish);
      };
      handle.addEventListener("pointermove", move);
      handle.addEventListener("pointerup", finish);
      handle.addEventListener("pointercancel", finish);
    });
  }
}
```

Call `event.preventDefault()` on pointerdown and close any open cell menu before dragging.

- [ ] **Step 4: Run syntax and focused contract checks**

```bash
node --check internal/logweb/assets/app.js
go test ./internal/logweb -run TestWorkspaceContainsResizableTableAndActionMenu -count=1
```

Expected: the focused Go test still FAILS only for Message/menu markers; JavaScript syntax exits 0.

### Task 3: Selectable Fields, Message Disclosure, and Menus

**Files:**
- Modify: `internal/logweb/assets/index.html`
- Modify: `internal/logweb/assets/app.css`
- Modify: `internal/logweb/assets/app.js`
- Test: `internal/logweb/server_test.go`

- [ ] **Step 1: Add the reusable menu host**

Place this after the workspace `main` and before `app.js`:

```html
<div id="cell-action-menu" class="cell-action-menu" role="menu" hidden>
  <button type="button" role="menuitem" data-action="filter">Add to filter</button>
  <button type="button" role="menuitem" data-action="copy">Copy full value</button>
</div>
```

Add `cellActionMenu` to `elements`. The script changes labels and hides the filter item for Message.

- [ ] **Step 2: Replace direct-click structured field buttons**

Change `appendFieldCell` so Search ID, User ID, and Source render the same structure:

```js
const valueText = document.createElement("span");
valueText.className = "field-value";
valueText.textContent = value;

const trigger = document.createElement("button");
trigger.type = "button";
trigger.className = "cell-action-trigger";
trigger.textContent = "⋮";
trigger.dataset.fieldTarget = target;
trigger.dataset.value = value;
trigger.setAttribute("aria-label", `Actions for ${target}`);
trigger.setAttribute("aria-haspopup", "menu");
trigger.setAttribute("aria-expanded", "false");
trigger.addEventListener("click", () => openCellActionMenu(trigger));

cell.className = "interactive-cell";
cell.append(valueText, trigger);
```

Remove the old field-value click handler completely. Level behavior remains unchanged.

- [ ] **Step 3: Replace the Message button with selectable content**

In `appendRecord`, render:

```js
const messageRow = document.createElement("div");
messageRow.className = "message-row";

const disclosure = document.createElement("button");
disclosure.type = "button";
disclosure.className = "message-disclosure";
disclosure.setAttribute("aria-expanded", "false");
disclosure.setAttribute("aria-label", "Expand message");
disclosure.textContent = "›";

const messageText = document.createElement("span");
messageText.className = "message-text";
messageText.textContent = record.message || record.raw || "";

const trigger = document.createElement("button");
trigger.type = "button";
trigger.className = "cell-action-trigger message-action-trigger";
trigger.textContent = "⋮";
trigger.dataset.message = messageText.textContent;
trigger.dataset.actionKind = "message";
trigger.setAttribute("aria-label", "Message actions");
trigger.setAttribute("aria-haspopup", "menu");
trigger.setAttribute("aria-expanded", "false");
trigger.addEventListener("click", () => openCellActionMenu(trigger));
```

Only `disclosure` toggles `.expanded`, `aria-expanded`, its label, and its glyph. Message text receives no click, pointerdown, or mouseup handler.

After inserting the row, call `updateMessageDisclosure(messageRow)`. The function hides the disclosure when collapsed text is not overflowed and contains no newline. When called without an argument, it remeasures every Message row after column resize.

- [ ] **Step 4: Style selectable text and zero-width hover actions**

Use `user-select: text` and `cursor: text` on `.field-value` and `.message-text`. Preserve one-line ellipsis while collapsed; `.message-row.expanded .message-text` uses `white-space: pre-wrap` and `overflow-wrap: anywhere`.

Make `.interactive-cell` and `.message-row` positioned containers. Make `.cell-action-trigger` absolute at the right edge with opacity 0 and no pointer events. Reveal it on cell hover, `:focus-visible`, while `aria-expanded="true"`, and under `@media (hover: none)`. This overlay must not add padding or a flex/grid track.

- [ ] **Step 5: Implement one fixed-position menu**

Implement these functions with a single menu owner:

```js
function openCellActionMenu(trigger) {
  closeCellActionMenu();
  state.actionMenuTrigger = trigger;
  trigger.setAttribute("aria-expanded", "true");
  const isMessage = trigger.dataset.actionKind === "message";
  const filterItem = elements.cellActionMenu.querySelector('[data-action="filter"]');
  const copyItem = elements.cellActionMenu.querySelector('[data-action="copy"]');
  filterItem.hidden = isMessage;
  copyItem.textContent = isMessage ? "Copy full message" : "Copy full value";
  elements.cellActionMenu.hidden = false;
  positionCellActionMenu(trigger);
  elements.cellActionMenu.querySelector('[role="menuitem"]:not([hidden])').focus();
}

function closeCellActionMenu({ restoreFocus = false } = {}) {
  const trigger = state.actionMenuTrigger;
  if (!trigger) return;
  elements.cellActionMenu.hidden = true;
  trigger.setAttribute("aria-expanded", "false");
  state.actionMenuTrigger = null;
  if (restoreFocus && trigger.isConnected) trigger.focus();
}
```

`positionCellActionMenu` uses `getBoundingClientRect()` and fixed coordinates, aligns the menu's right edge to the trigger, and flips above/left when it would cross the viewport. Because the menu is a direct body child, table-cell and scroll-container overflow cannot clip it.

- [ ] **Step 6: Implement menu commands and keyboard dismissal**

For Filter, map `fieldTarget` to `elements.searchID`, `elements.userID`, or `elements.source`, assign the full dataset value, focus the input, and do not run a search.

For Copy, call a helper that supports both secure contexts and the viewer's
common private-network HTTP deployment:

```js
async function copyText(value) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch (_) {
      // Fall through for private-network HTTP and denied permissions.
    }
  }
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.append(textarea);
  textarea.select();
  const copied = document.execCommand("copy");
  textarea.remove();
  if (!copied) throw new Error("copy command rejected");
}
```

The Copy command obtains the value from `trigger.dataset.message` or
`trigger.dataset.value` and passes it to `copyText`.

On success, close the menu and set `searchStatus` to a concise copied message. On failure, keep the menu open and set `searchStatus` to `Copy failed`.

Add document outside-click and Escape handlers, ArrowUp/ArrowDown cycling across visible menu items, and Enter activation through native button behavior. Close without restoring focus on table scroll, Clear, and row replacement; close with restored focus on Escape.

- [ ] **Step 7: Run focused and package verification**

```bash
gofmt -w internal/logweb/server_test.go
node --check internal/logweb/assets/app.js
go test ./internal/logweb -run TestWorkspaceContainsResizableTableAndActionMenu -count=1
go test ./internal/logweb -count=1
git diff --check
```

Expected: all commands PASS.

- [ ] **Step 8: Commit the feature**

```bash
git add internal/logweb/assets/index.html internal/logweb/assets/app.css internal/logweb/assets/app.js internal/logweb/server_test.go
git commit -m "feat: improve log table interactions"
```

### Task 4: Browser and Repository Verification

**Files:**
- Modify only if verification exposes a defect in the files above.

- [ ] **Step 1: Verify the real browser workflow**

Start an isolated demo log server and use Playwright CLI at desktop width and 390px mobile width. Verify:

- dragging every header handle changes only its intended column and honors its minimum;
- hidden action buttons reserve no visible width and appear on hover/focus;
- Search ID, User ID, and Source menus render above adjacent rows;
- Add to filter updates only its matching input and does not immediately search;
- Copy full value copies visually truncated data;
- copying succeeds from the private-network HTTP demo origin;
- collapsed Message text can be selected without expanding;
- the disclosure button alone expands and collapses;
- expanded Message permits partial text selection;
- Message actions appear in both states and copy the full value;
- menu placement flips near viewport edges;
- active Follow can insert a row while another row remains expanded; and
- browser console reports zero errors and warnings.

- [ ] **Step 2: Run complete verification**

```bash
gofmt -w internal/logweb
go vet ./...
go test ./... -count=1
go test -race ./internal/logweb ./internal/logview ./cmd -count=1
node --check internal/logweb/assets/app.js
git diff --check
```

Expected: all commands exit 0 with no test or race failures.

- [ ] **Step 3: Check repository hygiene**

Run the repository's sensitive-example scan using the existing split-literal convention. Remove `.playwright-cli`, `output`, and temporary demo directories. Confirm `git status --short` contains no generated artifacts and no unintended source changes.

- [ ] **Step 4: Commit verification fixes only when necessary**

If browser or full verification requires a source correction, repeat the relevant focused and complete commands before committing it. Do not create an empty commit when no correction is needed.
