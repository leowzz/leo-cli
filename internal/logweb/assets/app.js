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
};

const elements = {
  connection: document.querySelector("#connection"),
  range: document.querySelector("#range"),
  include: document.querySelector("#include"),
  exclude: document.querySelector("#exclude"),
  searchID: document.querySelector("#search-id"),
  userID: document.querySelector("#user-id"),
  source: document.querySelector("#source"),
  regex: document.querySelector("#regex"),
  caseSensitive: document.querySelector("#case-sensitive"),
  search: document.querySelector("#search"),
  cancel: document.querySelector("#cancel"),
  scanSize: document.querySelector("#scan-size"),
  fileTree: document.querySelector("#file-tree"),
  levels: document.querySelector("#levels"),
  searchStatus: document.querySelector("#search-status"),
  resultCount: document.querySelector("#result-count"),
  logScroll: document.querySelector("#log-scroll"),
  logBody: document.querySelector("#log-body"),
  emptyState: document.querySelector("#empty-state"),
  follow: document.querySelector("#follow"),
  followStatus: document.querySelector("#follow-status"),
  waitingCount: document.querySelector("#waiting-count"),
  discardedCount: document.querySelector("#discarded-count"),
  jumpLatest: document.querySelector("#jump-latest"),
};

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
  } catch (error) {
    setConnection("error", "Catalog unavailable");
    appendSystem(error.message || String(error), true);
  }
}

function selectRecentFiles() {
  const cutoff = Date.now() - Number(elements.range.value) * 1000;
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
  const seconds = Number(elements.range.value);
  const now = new Date();
  return {
    fileIds: [...state.selected],
    start: new Date(now.getTime() - seconds * 1000).toISOString(),
    end: now.toISOString(),
    include: parseTerms(elements.include.value),
    exclude: parseTerms(elements.exclude.value),
    regex: elements.regex.checked,
    caseSensitive: elements.caseSensitive.checked,
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
  const seconds = Number(elements.range.value);
  if (seconds > 86400 && !window.confirm(`Scan ${elements.scanSize.textContent}?`)) return;

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
    await consumeNDJSON(response.body, handleSearchEvent);
  } catch (error) {
    if (error.name === "AbortError") elements.searchStatus.textContent = "Cancelled";
    else {
      elements.searchStatus.textContent = "Search failed";
      appendSystem(error.message || String(error), true);
    }
  } finally {
    if (state.searchController === controller) state.searchController = null;
    elements.search.disabled = false;
    elements.cancel.disabled = true;
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
    const labels = { complete: "Search complete", limit: "Result limit reached", timeout: "Search timed out" };
    elements.searchStatus.textContent = labels[event.reason] || event.reason;
  }
}

function appendRecord(record, live) {
  const recordKey = `${record.fileId}:${record.offset}`;
  if (state.seen.has(recordKey)) return;
  state.seen.add(recordKey);
  const row = document.createElement("tr");
  row.dataset.recordKey = recordKey;
  appendCell(row, displayTime(record), "");
  appendLevelCell(row, record.level, record.fileName);
  appendFieldCell(row, record.searchId, "search-id");
  appendFieldCell(row, record.userId, "user-id");
  appendFieldCell(row, record.source, "source");

  const messageCell = document.createElement("td");
  const message = document.createElement("button");
  message.type = "button";
  message.className = "message-button";
  message.textContent = record.message || record.raw || "";
  message.title = record.truncated ? "Line truncated by server limit" : "Expand message";
  message.addEventListener("click", () => message.classList.toggle("expanded"));
  messageCell.append(message);
  row.append(messageCell);
  elements.logBody.append(row);
  state.records++;
  enforceRowLimit();
  updateRecordCount();
  elements.emptyState.hidden = true;

  if (live) {
    if (state.autoScroll) scrollLatest();
    else {
      state.waiting++;
      updateFollowCounters();
    }
  }
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
    const button = document.createElement("button");
    button.type = "button";
    button.className = "field-button";
    button.textContent = value;
    button.title = `Filter by ${value}`;
    button.addEventListener("click", () => {
      const filter = {
        "search-id": elements.searchID,
        "user-id": elements.userID,
        source: elements.source,
      }[target];
      filter.value = value;
      filter.focus();
    });
    cell.append(button);
  } else cell.textContent = "-";
  row.append(cell);
}

function appendCell(row, text, className) {
  const cell = document.createElement("td");
  cell.textContent = text || "-";
  if (className) cell.className = className;
  row.append(cell);
  return cell;
}

function appendSystem(message, warning) {
  const row = document.createElement("tr");
  row.className = warning ? "system-row warning-row" : "system-row";
  const cell = document.createElement("td");
  cell.colSpan = 6;
  cell.textContent = message;
  row.append(cell);
  elements.logBody.append(row);
  enforceRowLimit();
  elements.emptyState.hidden = true;
}

function clearRecords() {
  elements.logBody.replaceChildren();
  state.seen.clear();
  state.records = 0;
  state.waiting = 0;
  state.discarded = 0;
  elements.emptyState.hidden = false;
  updateRecordCount();
  updateFollowCounters();
}

function enforceRowLimit() {
  while (elements.logBody.children.length > MAX_ROWS) {
    const oldest = elements.logBody.firstElementChild;
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
      appendSystem(event.message, false);
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
    elements.logScroll.scrollTop = elements.logScroll.scrollHeight;
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

elements.search.addEventListener("click", runSearch);
elements.cancel.addEventListener("click", () => state.searchController?.abort());
elements.follow.addEventListener("click", startFollow);
elements.jumpLatest.addEventListener("click", scrollLatest);
elements.range.addEventListener("change", updateScanSize);
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
  const distance = elements.logScroll.scrollHeight - elements.logScroll.scrollTop - elements.logScroll.clientHeight;
  state.autoScroll = distance < 32;
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

loadCatalog();
