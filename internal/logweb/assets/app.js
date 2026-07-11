"use strict";

const MAX_ROWS = 1500;
const state = {
  files: [],
  selected: new Set(),
  searchController: null,
  eventSource: null,
  autoScroll: true,
  waiting: 0,
  discarded: 0,
  records: 0,
  seen: new Set(),
  rangeSeconds: 3600,
  clearedAt: null,
  arrivalSequence: 0,
  actionMenuTrigger: null,
};

const elements = {
  connection: document.querySelector("#connection"),
  rangeApply: document.querySelector("#range-apply"),
  rangeMenu: document.querySelector("#range-menu"),
  include: document.querySelector("#include"),
  exclude: document.querySelector("#exclude"),
  searchID: document.querySelector("#search-id"),
  userID: document.querySelector("#user-id"),
  source: document.querySelector("#source"),
  regex: document.querySelector("#regex"),
  caseSensitive: document.querySelector("#case-sensitive"),
  search: document.querySelector("#search"),
  cancel: document.querySelector("#cancel"),
  clear: document.querySelector("#clear"),
  scanSize: document.querySelector("#scan-size"),
  fileTree: document.querySelector("#file-tree"),
  levels: document.querySelector("#levels"),
  showUnparsed: document.querySelector("#show-unparsed"),
  searchStatus: document.querySelector("#search-status"),
  resultCount: document.querySelector("#result-count"),
  logScroll: document.querySelector("#log-scroll"),
  logTable: document.querySelector("#log-table"),
  logBody: document.querySelector("#log-body"),
  emptyState: document.querySelector("#empty-state"),
  cellActionMenu: document.querySelector("#cell-action-menu"),
  follow: document.querySelector("#follow"),
  followStatus: document.querySelector("#follow-status"),
  waitingCount: document.querySelector("#waiting-count"),
  discardedCount: document.querySelector("#discarded-count"),
  jumpLatest: document.querySelector("#jump-latest"),
};

function initColumnResizing() {
  for (const handle of elements.logTable.querySelectorAll(".column-resize")) {
    handle.addEventListener("pointerdown", (event) => {
      const column = elements.logTable.querySelector(`col[data-column="${handle.dataset.column}"]`);
      const startX = event.clientX;
      const startWidth = column.getBoundingClientRect().width;
      const minWidth = Number(column.dataset.minWidth);
      closeCellActionMenu();
      event.preventDefault();
      handle.setPointerCapture(event.pointerId);
      handle.classList.add("dragging");

      const move = (moveEvent) => {
        const width = Math.max(minWidth, startWidth + moveEvent.clientX - startX);
        column.style.width = `${width}px`;
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
  elements.cellActionMenu.hidden = true;
  if (!trigger) return;
  trigger.setAttribute("aria-expanded", "false");
  state.actionMenuTrigger = null;
  if (restoreFocus && trigger.isConnected) trigger.focus();
}

function positionCellActionMenu(trigger) {
  const gap = 4;
  const viewportPadding = 6;
  const triggerRect = trigger.getBoundingClientRect();
  elements.cellActionMenu.style.left = "0px";
  elements.cellActionMenu.style.top = "0px";
  const menuRect = elements.cellActionMenu.getBoundingClientRect();
  let left = triggerRect.right - menuRect.width;
  let top = triggerRect.bottom + gap;
  if (left < viewportPadding) left = Math.min(triggerRect.left, window.innerWidth - menuRect.width - viewportPadding);
  if (top + menuRect.height > window.innerHeight - viewportPadding) top = triggerRect.top - menuRect.height - gap;
  left = Math.max(viewportPadding, Math.min(left, window.innerWidth - menuRect.width - viewportPadding));
  top = Math.max(viewportPadding, Math.min(top, window.innerHeight - menuRect.height - viewportPadding));
  elements.cellActionMenu.style.left = `${left}px`;
  elements.cellActionMenu.style.top = `${top}px`;
}

async function copyText(value) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch (_) {
      // Fall through when clipboard permissions or the current origin reject access.
    }
  }
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.append(textarea);
  let copied = false;
  try {
    textarea.select();
    copied = document.execCommand("copy");
  } finally {
    textarea.remove();
  }
  if (!copied) throw new Error("copy command rejected");
}

async function loadCatalog() {
  try {
    const response = await fetch("/api/files", { cache: "no-store" });
    if (!response.ok) throw new Error(await response.text());
    const payload = await response.json();
    state.files = payload.files || [];
    selectRecentFiles();
    renderFileTree(payload.roots || []);
    updateScanSize();
    setConnection("online", `${state.files.length} files`);
    runSearch();
    startFollow();
  } catch (error) {
    setConnection("error", "Catalog unavailable");
    appendSystem(error.message || String(error), true);
  }
}

function selectRecentFiles() {
  const cutoff = Date.now() - state.rangeSeconds * 1000;
  state.selected.clear();
  for (const file of state.files) {
    if (new Date(file.modifiedAt).getTime() >= cutoff) state.selected.add(file.id);
  }
}

function renderFileTree(roots) {
  elements.fileTree.replaceChildren();
  const filesByRoot = new Map();
  for (const root of roots) filesByRoot.set(root, []);
  for (const file of state.files) {
    if (!filesByRoot.has(file.root)) filesByRoot.set(file.root, []);
    filesByRoot.get(file.root).push(file);
  }
  for (const [root, files] of filesByRoot) {
    const details = document.createElement("details");
    details.className = "tree-root";
    details.open = true;
    const summary = document.createElement("summary");
    summary.textContent = `${baseName(root)} (${files.length})`;
    summary.title = root;
    details.append(summary, renderTree(files));
    elements.fileTree.append(details);
  }
  if (!state.files.length) {
    const empty = document.createElement("div");
    empty.className = "file-name";
    empty.textContent = "No eligible log files";
    elements.fileTree.append(empty);
  }
}

function renderTree(files) {
  const root = { children: new Map(), files: [] };
  for (const file of files) {
    const parts = file.relativePath.split(/[\\/]/);
    let node = root;
    for (const part of parts.slice(0, -1)) {
      if (!node.children.has(part)) node.children.set(part, { children: new Map(), files: [] });
      node = node.children.get(part);
    }
    node.files.push(file);
  }
  return renderTreeNode(root, true);
}

function renderTreeNode(node, isRoot) {
  const list = document.createElement("ul");
  list.className = isRoot ? "tree-list root-list" : "tree-list";
  for (const [name, child] of [...node.children.entries()].sort(([a], [b]) => a.localeCompare(b))) {
    const item = document.createElement("li");
    item.className = "tree-directory";
    const label = document.createElement("span");
    label.textContent = name;
    item.append(label, renderTreeNode(child, false));
    list.append(item);
  }
  for (const file of node.files.sort((a, b) => a.name.localeCompare(b.name))) {
    const item = document.createElement("li");
    item.className = "file-item";
    const label = document.createElement("label");
    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.checked = state.selected.has(file.id);
    checkbox.dataset.fileId = file.id;
    checkbox.addEventListener("change", () => {
      if (checkbox.checked) state.selected.add(file.id);
      else state.selected.delete(file.id);
      updateScanSize();
    });
    const name = document.createElement("span");
    name.className = "file-name";
    name.textContent = file.name;
    name.title = file.relativePath;
    label.append(checkbox, name);
    item.append(label);
    list.append(item);
  }
  return list;
}

function updateFileCheckboxes() {
  for (const input of elements.fileTree.querySelectorAll("input[data-file-id]")) {
    input.checked = state.selected.has(input.dataset.fileId);
  }
  updateScanSize();
}

function updateScanSize() {
  const selectedFiles = state.files.filter((file) => state.selected.has(file.id));
  const bytes = selectedFiles.reduce((sum, file) => sum + file.size, 0);
  elements.scanSize.textContent = `${selectedFiles.length} files / ${formatBytes(bytes)}`;
}

function searchQuery() {
  const now = new Date();
  const start = state.clearedAt || new Date(now.getTime() - state.rangeSeconds * 1000);
  return {
    fileIds: [...state.selected],
    start: start.toISOString(),
    end: now.toISOString(),
    include: parseTerms(elements.include.value),
    exclude: parseTerms(elements.exclude.value),
    regex: elements.regex.checked,
    caseSensitive: elements.caseSensitive.checked,
    includeUnparsed: elements.showUnparsed.checked,
    levels: [...elements.levels.querySelectorAll("input:checked")].map((input) => input.value),
    searchIds: parseTerms(elements.searchID.value),
    userIds: parseTerms(elements.userID.value),
    sources: parseTerms(elements.source.value),
  };
}

async function runSearch() {
  if (state.searchController) state.searchController.abort();
  if (!state.selected.size) {
    clearRecords();
    appendSystem("Select at least one log file", true);
    return;
  }
  if (!state.clearedAt && state.rangeSeconds > 86400 && !window.confirm(`Scan ${elements.scanSize.textContent}?`)) return;

  clearRecords();
  const controller = new AbortController();
  state.searchController = controller;
  elements.search.disabled = true;
  elements.cancel.disabled = false;
  elements.searchStatus.textContent = "Preparing scan";
  try {
    const response = await fetch("/api/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(searchQuery()),
      signal: controller.signal,
    });
    if (!response.ok) throw new Error(await response.text());
    await consumeNDJSON(response.body, (event) => {
      if (state.searchController !== controller) return;
      handleSearchEvent(event);
    });
  } catch (error) {
    if (state.searchController !== controller) return;
    if (error.name === "AbortError") elements.searchStatus.textContent = "Cancelled";
    else {
      elements.searchStatus.textContent = "Search failed";
      appendSystem(error.message || String(error), true);
    }
  } finally {
    if (state.searchController === controller) {
      state.searchController = null;
      elements.search.disabled = false;
      elements.cancel.disabled = true;
    }
  }
}

async function consumeNDJSON(stream, onEvent) {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done });
    const lines = buffer.split("\n");
    buffer = lines.pop() || "";
    for (const line of lines) if (line.trim()) onEvent(JSON.parse(line));
    if (done) break;
  }
  if (buffer.trim()) onEvent(JSON.parse(buffer));
}

function handleSearchEvent(event) {
  if (event.type === "result" && event.record) appendRecord(event.record, false);
  if (event.type === "warning") appendSystem(event.warning, true);
  if (event.progress) {
    const progress = event.progress;
    elements.searchStatus.textContent = `${progress.scannedFiles}/${progress.candidateFiles} files, ${formatBytes(progress.scannedBytes)}/${formatBytes(progress.totalBytes)}`;
  }
  if (event.type === "done") {
    sortTimestampedRows();
    const labels = { complete: "Search complete", limit: "Result limit reached", timeout: "Search timed out" };
    elements.searchStatus.textContent = labels[event.reason] || event.reason;
  }
}

function appendRecord(record, live) {
  if (!record.parsed && !elements.showUnparsed.checked) return;
  const recordKey = `${record.fileId}:${record.offset}`;
  if (state.seen.has(recordKey)) return;
  state.seen.add(recordKey);
  const row = document.createElement("tr");
  row.dataset.recordKey = recordKey;
  row.dataset.arrival = String(++state.arrivalSequence);
  const timestamp = Date.parse(record.timestamp || "");
  if (!Number.isNaN(timestamp)) row.dataset.timestamp = String(timestamp);
  appendCell(row, displayTime(record), "");
  appendLevelCell(row, record.level, record.fileName);
  appendFieldCell(row, record.searchId, "search-id");
  appendFieldCell(row, record.userId, "user-id");
  appendFieldCell(row, record.source, "source");

  const messageCell = document.createElement("td");
  messageCell.className = "interactive-cell message-cell";
  if (record.truncated) messageCell.title = "Line truncated by server limit";
  const messageRow = document.createElement("div");
  messageRow.className = "message-row";
  const disclosure = document.createElement("button");
  disclosure.type = "button";
  disclosure.className = "message-disclosure";
  disclosure.hidden = true;
  disclosure.setAttribute("aria-expanded", "false");
  disclosure.setAttribute("aria-label", "Expand message");
  disclosure.textContent = ">";
  disclosure.addEventListener("click", () => {
    const expanded = messageRow.classList.toggle("expanded");
    disclosure.setAttribute("aria-expanded", String(expanded));
    disclosure.setAttribute("aria-label", expanded ? "Collapse message" : "Expand message");
    disclosure.textContent = expanded ? "v" : ">";
    updateMessageDisclosure(messageRow);
  });
  const messageText = document.createElement("span");
  messageText.className = "message-text";
  messageText.textContent = record.message || record.raw || "";
  const trigger = createCellActionTrigger("Message actions");
  trigger.classList.add("message-action-trigger");
  trigger.dataset.actionKind = "message";
  trigger.dataset.message = messageText.textContent;
  messageRow.append(disclosure, messageText, trigger);
  messageCell.append(messageRow);
  row.append(messageCell);
  insertNewestRow(row, live);
  updateMessageDisclosure(messageRow);
  state.records++;
  updateRecordCount();
  elements.emptyState.hidden = true;
}

function appendLevelCell(row, value, fileName) {
  const cell = document.createElement("td");
  cell.className = `level-cell level-${cssToken(value)}`;
  cell.title = fileName || "";
  if (value) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "field-button";
    button.textContent = value;
    button.title = `Filter by ${value}`;
    button.addEventListener("click", () => {
      for (const input of elements.levels.querySelectorAll("input")) {
        if (input.value === value.toUpperCase()) input.checked = true;
      }
    });
    cell.append(button);
  } else cell.textContent = "-";
  row.append(cell);
}

function appendFieldCell(row, value, target) {
  const cell = document.createElement("td");
  if (value) {
    cell.className = "interactive-cell";
    const valueText = document.createElement("span");
    valueText.className = "field-value";
    valueText.textContent = value;
    const trigger = createCellActionTrigger(`Actions for ${target}`);
    trigger.dataset.fieldTarget = target;
    trigger.dataset.value = value;
    cell.append(valueText, trigger);
  } else cell.textContent = "-";
  row.append(cell);
}

function createCellActionTrigger(label) {
  const trigger = document.createElement("button");
  trigger.type = "button";
  trigger.className = "cell-action-trigger";
  trigger.textContent = "\u22ee";
  trigger.setAttribute("aria-label", label);
  trigger.setAttribute("aria-haspopup", "menu");
  trigger.setAttribute("aria-expanded", "false");
  trigger.addEventListener("click", () => openCellActionMenu(trigger));
  return trigger;
}

function updateMessageDisclosure(messageRow) {
  const rows = messageRow ? [messageRow] : elements.logBody.querySelectorAll(".message-row");
  for (const row of rows) {
    const disclosure = row.querySelector(".message-disclosure");
    const messageText = row.querySelector(".message-text");
    if (row.classList.contains("expanded")) {
      disclosure.hidden = false;
      continue;
    }
    disclosure.hidden = true;
    const multiline = messageText.textContent.includes("\n");
    const overflowed = messageText.scrollWidth > messageText.clientWidth + 1;
    disclosure.hidden = !multiline && !overflowed;
  }
}

function appendCell(row, text, className) {
  const cell = document.createElement("td");
  cell.textContent = text || "-";
  if (className) cell.className = className;
  row.append(cell);
  return cell;
}

function appendSystem(message, warning, live = false) {
  const row = document.createElement("tr");
  row.className = warning ? "system-row warning-row" : "system-row";
  row.dataset.arrival = String(++state.arrivalSequence);
  const cell = document.createElement("td");
  cell.colSpan = 6;
  cell.textContent = message;
  row.append(cell);
  insertNewestRow(row, live);
  elements.emptyState.hidden = true;
}

function insertNewestRow(row, live) {
  const wasAtTop = state.autoScroll;
  row.dataset.live = live ? "true" : "false";
  if (live) elements.logBody.prepend(row);
  else insertHistoricalRow(row);
  const insertedHeight = row.getBoundingClientRect().height;
  enforceRowLimit();
  if (!live) return;
  if (wasAtTop) {
    scrollLatest();
    return;
  }
  elements.logScroll.scrollTop += insertedHeight;
  state.waiting++;
  updateFollowCounters();
}

function insertHistoricalRow(row) {
  let insertionPoint = elements.logBody.firstElementChild;
  while (insertionPoint?.dataset.live === "true") insertionPoint = insertionPoint.nextElementSibling;
  elements.logBody.insertBefore(row, insertionPoint);
}

function sortTimestampedRows() {
  closeCellActionMenu();
  const rows = [...elements.logBody.children];
  const timestamped = rows
    .filter((row) => row.dataset.live !== "true" && row.dataset.timestamp)
    .sort((left, right) => Number(right.dataset.timestamp) - Number(left.dataset.timestamp));
  let timestampIndex = 0;
  const ordered = rows.map((row) => row.dataset.live !== "true" && row.dataset.timestamp ? timestamped[timestampIndex++] : row);
  elements.logBody.replaceChildren(...ordered);
}

function clearRecords() {
  closeCellActionMenu();
  elements.logBody.replaceChildren();
  state.seen.clear();
  state.records = 0;
  state.waiting = 0;
  state.discarded = 0;
  state.arrivalSequence = 0;
  elements.emptyState.hidden = false;
  updateRecordCount();
  updateFollowCounters();
}

function enforceRowLimit() {
  while (elements.logBody.children.length > MAX_ROWS) {
    const oldest = elements.logBody.lastElementChild;
    if (oldest.contains(state.actionMenuTrigger)) closeCellActionMenu();
    if (oldest.dataset.recordKey) state.seen.delete(oldest.dataset.recordKey);
    oldest.remove();
    state.discarded++;
  }
  updateFollowCounters();
}

function updateRecordCount() {
  elements.resultCount.textContent = `${state.records} ${state.records === 1 ? "record" : "records"}`;
}

function startFollow() {
  if (state.eventSource) {
    stopFollow();
    return;
  }
  if (!state.selected.size) {
    appendSystem("Select at least one log file", true);
    return;
  }
  const source = new EventSource(`/api/follow?files=${encodeURIComponent([...state.selected].join(","))}`);
  state.eventSource = source;
  elements.follow.classList.add("active");
  elements.follow.textContent = "Stop follow";
  elements.followStatus.textContent = `Following ${state.selected.size} files`;
  source.onopen = () => { elements.followStatus.textContent = `Live / ${state.selected.size} files`; };
  source.onmessage = (message) => {
    const event = JSON.parse(message.data);
    if (event.type === "record" && event.record) appendRecord(event.record, true);
    if (event.type === "system") {
      if (/rotated|truncated/.test(event.message || "")) clearSeenForFile(event.fileId);
      appendSystem(event.message, false, true);
    }
  };
  source.onerror = () => {
    elements.followStatus.textContent = source.readyState === EventSource.CLOSED ? "Follow stopped" : "Reconnecting";
  };
}

function stopFollow() {
  if (state.eventSource) state.eventSource.close();
  state.eventSource = null;
  elements.follow.classList.remove("active");
  elements.follow.textContent = "Start follow";
  elements.followStatus.textContent = "Follow stopped";
}

function updateFollowCounters() {
  elements.waitingCount.hidden = state.waiting === 0;
  elements.waitingCount.textContent = `${state.waiting} new`;
  elements.discardedCount.hidden = state.discarded === 0;
  elements.discardedCount.textContent = `${state.discarded} discarded`;
  elements.jumpLatest.disabled = state.waiting === 0;
}

function scrollLatest() {
  requestAnimationFrame(() => {
    elements.logScroll.scrollTop = 0;
    state.autoScroll = true;
    state.waiting = 0;
    updateFollowCounters();
  });
}

function setConnection(status, text) {
  elements.connection.className = `connection ${status}`;
  elements.connection.lastElementChild.textContent = text;
}

function parseTerms(value) {
  return value.split(/[\n,]/).map((term) => term.trim()).filter(Boolean);
}

function displayTime(record) {
  if (!record.timestamp) return record.timeText || "-";
  const date = new Date(record.timestamp);
  if (Number.isNaN(date.getTime())) return record.timeText || record.timestamp;
  return date.toLocaleString(undefined, { hour12: false });
}

function formatBytes(value) {
  if (!value) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const scaled = value / (1024 ** index);
  return `${scaled >= 10 || index === 0 ? scaled.toFixed(0) : scaled.toFixed(1)} ${units[index]}`;
}

function baseName(path) {
  const parts = path.split(/[\\/]/).filter(Boolean);
  return parts[parts.length - 1] || path;
}

function cssToken(value) {
  return String(value || "").replace(/[^A-Za-z0-9_-]/g, "").toUpperCase();
}

function clearSeenForFile(fileID) {
  const prefix = `${fileID}:`;
  for (const key of state.seen) if (key.startsWith(prefix)) state.seen.delete(key);
}

function selectedRangeLabel() {
  return elements.rangeMenu.options[elements.rangeMenu.selectedIndex].textContent;
}

function applySelectedRange() {
  state.rangeSeconds = Number(elements.rangeMenu.value);
  state.clearedAt = null;
  elements.rangeApply.textContent = selectedRangeLabel();
  elements.rangeApply.setAttribute("aria-label", `Apply ${selectedRangeLabel()} range`);
  runSearch();
}

function clearConsole() {
  const activeSearch = state.searchController;
  state.searchController = null;
  activeSearch?.abort();
  state.clearedAt = new Date();
  clearRecords();
  elements.search.disabled = false;
  elements.cancel.disabled = true;
  elements.searchStatus.textContent = `Cleared at ${state.clearedAt.toLocaleTimeString(undefined, { hour12: false })}`;
}

elements.cellActionMenu.addEventListener("click", async (event) => {
  const item = event.target.closest('[role="menuitem"]');
  const trigger = state.actionMenuTrigger;
  if (!item || !trigger) return;
  if (item.dataset.action === "filter") {
    const filter = {
      "search-id": elements.searchID,
      "user-id": elements.userID,
      source: elements.source,
    }[trigger.dataset.fieldTarget];
    if (!filter) return;
    filter.value = trigger.dataset.value;
    closeCellActionMenu();
    filter.focus();
    return;
  }
  if (item.dataset.action === "copy") {
    const isMessage = trigger.dataset.actionKind === "message";
    const value = isMessage ? trigger.dataset.message : trigger.dataset.value;
    try {
      await copyText(value);
      if (state.actionMenuTrigger === trigger) closeCellActionMenu();
      elements.searchStatus.textContent = isMessage ? "Message copied" : "Value copied";
    } catch (_) {
      elements.searchStatus.textContent = "Copy failed";
      if (state.actionMenuTrigger === trigger) item.focus();
    }
  }
});

elements.cellActionMenu.addEventListener("keydown", (event) => {
  if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
  const items = [...elements.cellActionMenu.querySelectorAll('[role="menuitem"]:not([hidden])')];
  if (!items.length) return;
  event.preventDefault();
  const current = items.indexOf(document.activeElement);
  const direction = event.key === "ArrowDown" ? 1 : -1;
  const next = current < 0 ? 0 : (current + direction + items.length) % items.length;
  items[next].focus();
});

document.addEventListener("pointerdown", (event) => {
  if (!state.actionMenuTrigger) return;
  if (elements.cellActionMenu.contains(event.target) || state.actionMenuTrigger.contains(event.target)) return;
  closeCellActionMenu();
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && state.actionMenuTrigger) {
    event.preventDefault();
    closeCellActionMenu({ restoreFocus: true });
  }
});

elements.search.addEventListener("click", runSearch);
elements.cancel.addEventListener("click", () => state.searchController?.abort());
elements.clear.addEventListener("click", clearConsole);
elements.rangeApply.addEventListener("click", applySelectedRange);
elements.rangeMenu.addEventListener("change", applySelectedRange);
elements.showUnparsed.addEventListener("change", runSearch);
elements.follow.addEventListener("click", startFollow);
elements.jumpLatest.addEventListener("click", scrollLatest);
elements.levels.addEventListener("change", () => {});
document.querySelector("#select-all").addEventListener("click", () => {
  state.selected = new Set(state.files.map((file) => file.id));
  updateFileCheckboxes();
});
document.querySelector("#select-none").addEventListener("click", () => {
  state.selected.clear();
  updateFileCheckboxes();
});
elements.logScroll.addEventListener("scroll", () => {
  closeCellActionMenu();
  state.autoScroll = elements.logScroll.scrollTop < 32;
  if (state.autoScroll && state.waiting) {
    state.waiting = 0;
    updateFollowCounters();
  }
});
for (const input of [elements.include, elements.exclude, elements.searchID, elements.userID, elements.source]) {
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") runSearch();
  });
}
window.addEventListener("beforeunload", () => {
  state.searchController?.abort();
  state.eventSource?.close();
});
window.addEventListener("resize", () => closeCellActionMenu());

initColumnResizing();
loadCatalog();
