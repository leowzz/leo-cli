# Log Viewer Table Interaction Design

**Date:** 2026-07-11

## Summary

Improve the log table for repeated inspection and copying workflows. Columns
become resizable, Message content remains selectable in both collapsed and
expanded states, and structured fields use an explicit action menu instead of
performing a filter action on the first click.

The table remains dense and newest-first. These changes affect only browser
interaction and presentation; search, parsing, Follow, Clear, and record
ordering contracts remain unchanged.

## Resizable Columns

Every column boundary except the final Message edge gains a resize handle in
the sticky table header.

- Dragging a handle changes the corresponding `<col>` width.
- Each column has a minimum width so content and controls cannot collapse into
  one another.
- The table may grow wider than the viewport; the existing horizontal table
  scroll remains the overflow mechanism.
- Resizing does not alter row height or interrupt Follow.
- Widths last for the current page session. Reloading restores the product
  defaults.
- Handles use `col-resize`, provide a visible hover/drag state, and support
  pointer input so mouse and touch follow the same code path.

The action buttons described below are absolutely positioned overlays. Hidden
buttons therefore consume no column width and do not cause text to reflow when
they appear.

## Structured Field Actions

Search ID, User ID, and Source use the same interaction.

The field value is plain selectable text. Clicking or dragging on the value
never changes filters. Hovering the field cell reveals a `More actions`
button at the right edge. The button is hidden otherwise, consumes no layout
width, and remains visible while focused or while its menu is open.

The menu contains:

1. **Add to filter**: place the complete value in the corresponding Search ID,
   User ID, or Source query input and focus that input. This preserves the
   existing behavior after the user explicitly chooses the command; it does
   not automatically run a search.
2. **Copy full value**: copy the complete underlying value, including text that
   is visually truncated in the cell.

Filter is the first menu item because it is the higher-priority Source
workflow, but opening the menu is the required confirmation step that prevents
accidental filtering.

Only one menu may be open at a time. Clicking outside it, pressing Escape,
starting a table scroll, or clearing/replacing rows closes it. The menu must
render above later table rows and outside cell clipping. Near the bottom or
right viewport edge, it repositions to remain visible.

## Message Interaction

Message is intentionally different from structured fields.

### Selection

Message text is a selectable text element, not a button. It handles no click
or pointer action. Users can drag-select and copy visible text while the
message is collapsed or any subset of text while it is expanded.

### Expand and Collapse

A dedicated disclosure button at the left edge is the only control that
changes Message layout.

- Long or multiline content shows the disclosure control.
- Short content that is fully visible does not show an unnecessary control.
- Activating the control expands the Message with preserved line breaks and
  wrapping.
- In expanded state the same control changes to Collapse and returns the row
  to its one-line state.
- Selecting text never expands or collapses a row.

The disclosure button exposes `aria-expanded` and an accessible label for its
current action.

### Message Actions

Hovering the Message cell reveals a `More actions` button at its right edge.
It follows the same overlay, focus persistence, menu dismissal, and clipping
rules as structured field actions. It is available in collapsed and expanded
states.

The initial Message menu contains **Copy full message**. Partial copying uses
native text selection. The menu provides a stable extension point for future
special copy commands without adding permanent inline buttons.

## Menu and Clipboard Feedback

Menu commands provide concise status feedback in the existing result/status
surface or an unobtrusive transient status element.

Copy commands prefer the Clipboard API when it is available. Because the log
viewer is commonly opened from a private-network HTTP address that is not a
secure browser context, they fall back to selecting a temporary off-screen
textarea and invoking the browser copy command. The temporary element is
always removed and the user's existing selection is not used as the source.

Clipboard failures do not alter filters or Message state. The UI reports the
failure and leaves the menu open so the user can fall back to native text
selection. Successful commands close the menu.

Keyboard behavior follows standard menu expectations:

- Enter or Space opens the action menu.
- Arrow keys move between menu items.
- Enter activates the focused item.
- Escape closes the menu and returns focus to its trigger.
- Tab leaves the menu through normal focus order.

## Responsive Behavior

The desktop table keeps its current dense dimensions. On narrow viewports the
table retains a stable minimum width and scrolls horizontally; columns do not
compress below their minimums. Resize and action controls remain attached to
their column or cell while scrolling.

Hover-only discovery is supplemented by keyboard focus. Devices without hover
show the overlaid action button at low emphasis so the command remains
discoverable; it still reserves no column width. Tapping it opens the menu
without triggering field filtering or Message expansion.

## Implementation Boundary

This is an embedded browser-asset change:

- `internal/logweb/assets/index.html` adds semantic header handles where
  needed.
- `internal/logweb/assets/app.css` defines resizable columns, selectable
  Message states, overlay action triggers, and menus.
- `internal/logweb/assets/app.js` owns resizing, disclosure state, reusable
  field/Message menus, clipboard commands, filtering commands, dismissal, and
  keyboard behavior.
- `internal/logweb/server_test.go` asserts the embedded controls and core
  interaction hooks.

No Go API or log record schema changes are required.

## Verification

Automated coverage will assert:

- all resizable headers expose handles and minimum widths;
- resizing changes only the intended `<col>`;
- Search ID, User ID, and Source render selectable text with identical menus;
- field clicks do not mutate filters;
- Add to filter changes only the corresponding input;
- Copy full value uses the untruncated value;
- Message text has no expand/collapse click handler;
- only the disclosure button toggles Message state;
- the Message menu is available in both states;
- action buttons do not reserve width while hidden; and
- menus close and return focus through the defined dismissal paths.

Playwright verification will cover desktop and mobile-width layouts, column
dragging, partial Message selection before and after expansion, structured
field filtering and copying, Message copying, menu overlap with adjacent rows,
Follow updates while a row is expanded, and browser console errors.
